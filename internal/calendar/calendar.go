package calendar

import (
	"context"	
	"core-regulus-backend/internal/db"
	"encoding/json"
	"fmt"
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
	Date string     `json:"date"`
	Slots []Interval `json:"slots"`
}

type DaysResponse struct {
	Days []TimeSlot `json:"days"`
}

type FreeSlot struct {
	TimeStart time.Time
	TimeEnd time.Time
}

type CalendarDaysInput struct {
	DateEnd time.Time `json:"dateEnd"`
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

func PrintFreeSlots() {	
	calendarService, calendarId := getService()
	
	from := time.Now()
	to := from.Add(8 * time.Hour)
	slotLen := 30 * time.Minute

	slots, err := GetFreeSlots(calendarService, calendarId, from, to, slotLen)
	if err != nil {
		log.Fatalf("Ошибка получения свободных слотов: %v", err)
	}

	fmt.Println("Свободные слоты:")
	for _, slot := range slots {
		fmt.Printf("- %s до %s\n", slot.TimeStart.Format("2006-01-02 15:04"), slot.TimeEnd.Format("2006-01-02 15:04"))
	}
}

func getTimeSlots(pool *pgxpool.Pool, interval CalendarDaysInput) ([]TimeSlot, error) {
	ctx := context.Background()
	
	var jsonData []byte
	
	query := "select service.get_free_slots('now()', $1)"
	err := pool.QueryRow(ctx, query, interval.DateEnd).Scan(&jsonData)
	if err != nil {
		return nil, err
	}

	var slots []TimeSlot
	if err := json.Unmarshal(jsonData, &slots); err != nil {
		return nil, err
	}	
	return slots, nil
}

func getService() (*calendar.Service, string) {
	calendarOnce.Do(func () {
		dbConfig := *db.Config()
		var ok bool
		calendarId, ok = dbConfig.GetString("googleCalendarId")
		if (!ok) {
			log.Fatal("googleCalendarId is not in config.config table")
		}
			
		var serviceData string
		serviceData, ok = dbConfig.GetString("googleCalendar")
		if (!ok) {
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

func InitRoutes(app *fiber.App) {
	app.Post("/calendar/days", func(c *fiber.Ctx) error {		
		var interval CalendarDaysInput
		if err := c.BodyParser(&interval); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot parse JSON",
			})
		}
		
		pool := db.Connect()
		timeSlots, dberr := getTimeSlots(pool, interval)
		if (dberr != nil) {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": dberr.Error(),
			})
		}

		response := DaysResponse{
			Days: timeSlots,
		}

		return c.JSON(response)
	})
}