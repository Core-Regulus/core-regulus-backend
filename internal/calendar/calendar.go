package calendar

import (
	"context"
	"core-regulus-backend/internal/db"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Interval struct {
	TimeStart string `json:"timeStart"`
	TimeEnd   string `json:"timeEnd"`
}

type TimeSlot struct {
	Date  string     `json:"date"`
	Slots []Interval `json:"slots"`
}

type DaysResponse struct {
	Days []TimeSlot `json:"days"`
}

type FreeSlot struct {
	TimeStart time.Time
	TimeEnd   time.Time
}

type CalendarDaysInput struct {
	DateStart string `json:"dateStart"`
	DateEnd   string `json:"dateEnd"`
}

type TimeSlotRecord struct {
	Id        string        `json:"id"`
	DayOfWeek string        `json:"dayOfWeek"`
	TimeStart string        `json:"timeStart"`
	Duration  time.Duration `json:"duration"`
	Attendees	[]Attendee		`json:"attendees"`
}

type Attendee struct {
	Name      string        `json:"name"`
	Email			string				`json:"email"`
}


var calendarService *calendar.Service
var calendarId string
var calendarOnce sync.Once

func GetFreeSlots(srv *calendar.Service, calendarID string, from, to time.Time, slotLength time.Duration) ([]FreeSlot, error) {
	ctx := context.Background()

	req := &calendar.FreeBusyRequest{
		TimeMin: from.Format(time.RFC3339),
		TimeMax: to.Format(time.RFC3339),
		Items: []*calendar.FreeBusyRequestItem{
			{Id: calendarID},
		},
	}

	resp, err := srv.Freebusy.Query(req).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	busy := resp.Calendars[calendarID].Busy

	var freeSlots []FreeSlot
	cursor := from

	for _, b := range busy {
		start, _ := time.Parse(time.RFC3339, b.Start)
		if cursor.Before(start) {
			for cursor.Add(slotLength).Before(start) || cursor.Add(slotLength).Equal(start) {
				freeSlots = append(freeSlots, FreeSlot{
					TimeStart: cursor,
					TimeEnd:   cursor.Add(slotLength),
				})
				cursor = cursor.Add(slotLength)
			}
		}
		if end, _ := time.Parse(time.RFC3339, b.End); cursor.Before(end) {
			cursor = end
		}
	}

	for cursor.Add(slotLength).Before(to) || cursor.Add(slotLength).Equal(to) {
		freeSlots = append(freeSlots, FreeSlot{
			TimeStart: cursor,
			TimeEnd:   cursor.Add(slotLength),
		})
		cursor = cursor.Add(slotLength)
	}

	return freeSlots, nil
}

func getTimeSlots(pool *pgxpool.Pool, from, to time.Time) ([]TimeSlot, error) {
	ctx := context.Background()

	var jsonData []byte

	err := pool.QueryRow(ctx, "select service.get_free_slots($1, $2);", from, to).Scan(&jsonData)
	if err != nil {
		return nil, err
	}

	var slots []TimeSlot
	if err := json.Unmarshal(jsonData, &slots); err != nil {
		return nil, err
	}

	var result []TimeSlot
	for _, slot := range slots {
		slotDate, err := time.Parse("2006-01-02", slot.Date)
		if err != nil {
			continue
		}
		if slotDate.After(from) && slotDate.Before(to) || slotDate.Equal(from) || slotDate.Equal(to) {
			result = append(result, slot)
		}
	}
	return result, nil
}

func getTargetSlot(pool *pgxpool.Pool, from time.Time) (*TimeSlotRecord, error) {
	ctx := context.Background()

	var jsonData []byte

	err := pool.QueryRow(ctx, "select service.get_target_slot($1)", from).Scan(&jsonData)
	if err != nil {
		return nil, err
	}

	if len(jsonData) == 0 {
		return nil, fmt.Errorf("slot is not found at %s", from.Format(time.RFC3339))
	}

	var slot TimeSlotRecord
	if err := json.Unmarshal(jsonData, &slot); err != nil {
		return nil, err
	}
	return &slot, nil
}

func getService() (*calendar.Service, string) {
	calendarOnce.Do(func() {
		dbConfig := *db.Config()
		var ok bool
		calendarId, ok = dbConfig.GetString("googleCalendarId")
		if !ok {
			log.Fatal("googleCalendarId is not in config.config table")
		}

		var serviceData string
		serviceData, ok = dbConfig.GetString("googleCalendar")
		if !ok {
			log.Fatal("googleCalendar is not in config.config table")
		}

		var creds *jwt.Config
		ctx := context.Background()

		creds, err := google.JWTConfigFromJSON([]byte(serviceData), calendar.CalendarScope)
		if err != nil {
			log.Fatalf("Can't parse JWT config: %v", err)
		}

		creds.Subject = calendarId
		client := creds.Client(ctx)

		calendarService, err = calendar.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			log.Fatalf("Can't initialize Calendar API client: %v", err)
		}
	})
	return calendarService, calendarId
}

func GetBusySlots(srv *calendar.Service, calendarID string, from, to time.Time) ([]FreeSlot, error) {
	ctx := context.Background()

	req := &calendar.FreeBusyRequest{
		TimeMin: from.Format(time.RFC3339),
		TimeMax: to.Format(time.RFC3339),
		Items: []*calendar.FreeBusyRequestItem{
			{Id: calendarID},
		},
	}

	resp, err := srv.Freebusy.Query(req).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	busy := resp.Calendars[calendarID].Busy

	var busySlots []FreeSlot

	for _, b := range busy {
		start, _ := time.Parse(time.RFC3339, b.Start)
		end, _ := time.Parse(time.RFC3339, b.End)
		busySlots = append(busySlots, FreeSlot{
			TimeStart: start,
			TimeEnd:   end,
		})
	}
	return busySlots, nil
}

func postCalendarDaysHandler(c *fiber.Ctx) error {
	var tInterval CalendarDaysInput

	if err := c.BodyParser(&tInterval); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	calendar, calendarId := getService()
	from, err := time.Parse("2006-01-02", tInterval.DateStart)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse timeStart",
		})
	}

	to, err := time.Parse("2006-01-02", tInterval.DateEnd)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse timeEnd",
		})
	}

	timeSlots, err := GetBusySlots(calendar, calendarId, from, to)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Cannot get busy slots from calendar",
		})
	}

	pool := db.Connect()
	timetable, dberr := getTimeSlots(pool, from, to)
	if dberr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": dberr.Error(),
		})
	}

	var result []TimeSlot
	for _, tt := range timetable {
		var slots []Interval
		for _, ts := range tt.Slots {
			start, _ := time.Parse("2006-01-02T15:04:05", ts.TimeStart)
			end, _ := time.Parse("2006-01-02T15:04:05", ts.TimeEnd)
			if checkTimeSlot(timeSlots, start, end) {
				continue
			}
			slots = append(slots, ts)
		}
		tmSt := TimeSlot{
			Date:  tt.Date,
			Slots: slots,
		}
		result = append(result, tmSt)
	}

	response := DaysResponse{
		Days: result,
	}

	return c.JSON(response)
}

func checkTimeSlot(busySlots []FreeSlot, start, end time.Time) bool {
	for _, tb := range busySlots {
		if start.After(tb.TimeStart) && start.Before(tb.TimeEnd) {
			return true
		}
		if start.Before(tb.TimeStart) && end.After(tb.TimeStart) {
			return true
		}
		if start.After(tb.TimeStart) && start.Before(tb.TimeEnd) {
			return true
		}
	}
	return false
}

type NewEventRequest struct {
	Time        string `json:"time"`
	Event       string `json:"eventName"`
	Email       string `json:"guestEmail"`
	Name        string `json:"guestName"`
	Description string `json:"guestDescription,omitempty"`
}

func CalendarConflictCheck(startTime time.Time, endTime time.Time) error {
	srv, calendarId := getService()	
	conflictCheck, err := calendar.NewEventsService(srv).List(calendarId).
	TimeMin(startTime.Format(time.RFC3339)).
	TimeMax(endTime.Format(time.RFC3339)).
	SingleEvents(true).
	OrderBy("startTime").
	Do()

	if err != nil {
		return err;
	}
	
	if len(conflictCheck.Items) > 0 {
		return errors.New("slot is busy; please choose another slot")
	}

	return nil
}

func postCalendarEventHandler(c *fiber.Ctx) error {
	var eventRequest NewEventRequest

	if err := c.BodyParser(&eventRequest); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	srv, calendarId := getService()

	startTime, err := time.Parse(time.RFC3339, eventRequest.Time)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid time format",
		})
	}
	pool := db.Connect()
	tsr, err := getTargetSlot(pool, startTime)
	if tsr == nil {
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":  "Time Slot Error",
				"reason": err.Error(),
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Time Slot is not found",
		})
	}
	
	endTime := startTime.Add(tsr.Duration * time.Second)
	err = CalendarConflictCheck(startTime, endTime)
	if (err != nil) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":  "Slot is busy",
			"reason": err.Error(),
		})
	}

	var eventAttendees []*calendar.EventAttendee

	eventAttendees = append(eventAttendees,
		&calendar.EventAttendee{
			Email:       eventRequest.Email,
			DisplayName: eventRequest.Name,
		},
	)
		
	for _, a := range tsr.Attendees {
		eventAttendees = append(eventAttendees, &calendar.EventAttendee{
			Email:       a.Email,
			DisplayName: a.Name,
		})
	}

	event := &calendar.Event{
		Summary:     eventRequest.Name,
		Description: eventRequest.Description,
		Status:      "tentative",
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
			TimeZone: "Europe/Belgrade",
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
			TimeZone: "Europe/Belgrade",
		},
		Attendees: eventAttendees,	
		ConferenceData: &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId: uuid.New().String(),
			},
		},
	}

	res := calendar.NewEventsService(srv).Insert(calendarId, event).SendUpdates("all").ConferenceDataVersion(1)
	createdEvent, err := res.Do()
	if err != nil {
		log.Printf("Google API error: %#v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":  "Unable to set up meeting",
			"reason": err.Error(),
		})
	}

	return c.JSON(createdEvent)
}

func InitRoutes(app *fiber.App) {
	app.Post("/calendar/days", postCalendarDaysHandler)
	app.Post("/calendar/event", postCalendarEventHandler)
}
