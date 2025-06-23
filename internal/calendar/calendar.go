package calendar

import (
	"context"
	"core-regulus-backend/internal/config"
	"core-regulus-backend/internal/db"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/calendar/v3"
)

type Interval struct {
	TimeStart string `json:"timeStart"`
	TimeEnd   string `json:"timeEnd"`
}

type TimeSlot struct {
	Date string     `json:"date"`
	Slots []Interval `json:"slots"`
}

type FreeSlot struct {
	TimeStart time.Time
	TimeEnd time.Time
}

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
	cfg := config.Get();		
	from := time.Now()
	to := from.Add(8 * time.Hour)
	slotLen := 30 * time.Minute

	slots, err := GetFreeSlots(cfg.Calendar.Service, cfg.Calendar.Id, from, to, slotLen)
	if err != nil {
		log.Fatalf("Ошибка получения свободных слотов: %v", err)
	}

	fmt.Println("Свободные слоты:")
	for _, slot := range slots {
		fmt.Printf("- %s до %s\n", slot.TimeStart.Format("2006-01-02 15:04"), slot.TimeEnd.Format("2006-01-02 15:04"))
	}
}

func getTimeSlots(pool *pgxpool.Pool) ([]TimeSlot, error) {
	ctx := context.Background()
	
	var jsonData []byte

	err := pool.QueryRow(ctx, "select service.get_free_slots(now(), now() + interval '1 month');").Scan(&jsonData)
	if err != nil {
		return nil, err
	}

	var slots []TimeSlot
	if err := json.Unmarshal(jsonData, &slots); err != nil {
		return nil, err
	}	
	return slots, nil
}

func InitRoutes(app *fiber.App) {
	app.Post("/calendar/days", func(c *fiber.Ctx) error {		
		var slots FreeSlot

		if err := c.BodyParser(&slots); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot parse JSON",
			})
		}
		
		pool := db.Connect()
		timeSlots, dberr := getTimeSlots(pool);
		if (dberr != nil) {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": dberr.Error(),
			})
		}

		PrintFreeSlots()

		return c.JSON(fiber.Map{
			"slots": timeSlots,
		})
	})
}