package calendar

import (
	"context"
	"core-regulus-backend/internal/db"
	"encoding/json"	
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2/google"
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
	DateStart time.Time `json:"dateStart"`
	DateEnd   time.Time `json:"dateEnd"`
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

		var creds *google.Credentials
		ctx := context.Background()
		creds, err := google.CredentialsFromJSON(ctx, []byte(serviceData), calendar.CalendarReadonlyScope)
		if err != nil {
			log.Fatalf("Can't load calendar account credentials: %v", err)
		}

		calendarService, err = calendar.NewService(ctx, option.WithCredentials(creds))
		if err != nil {
			log.Fatalf("Can't initialize Calendar API client: %v", err)
		}
	})
	return calendarService, calendarId
}

func GetBusySlots(srv *calendar.Service, calendarID string, from, to time.Time, slotLength time.Duration) ([]FreeSlot, error) {
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
	var tInterval Interval

	if err := c.BodyParser(&tInterval); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	calendar, calendarId := getService()
	from, err := time.Parse("2006-01-02", tInterval.TimeStart)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse timeStart",
		})
	}

	to, err := time.Parse("2006-01-02", tInterval.TimeEnd)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse timeEnd",
		})
	}
	pool := db.Connect()

	timeSlots, err := GetBusySlots(calendar, calendarId, from, to, 30*time.Minute)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Cannot get busy slots from calendar",
		})
	}

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

func InitRoutes(app *fiber.App) {
	app.Post("/calendar/days", postCalendarDaysHandler)
}
