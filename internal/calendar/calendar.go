package calendar

import (
	"context"
	"core-regulus-backend/internal/config"
	"fmt"
	"log"
	"time"	
	"google.golang.org/api/calendar/v3"	
)

type FreeSlot struct {
	Start time.Time
	End   time.Time
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
		return nil, fmt.Errorf("freebusy request error: %w", err)
	}
	
	busy := resp.Calendars[calendarID].Busy

	var freeSlots []FreeSlot
	cursor := from

	for _, b := range busy {
		start, _ := time.Parse(time.RFC3339, b.Start)
		if cursor.Before(start) {
			for cursor.Add(slotLength).Before(start) || cursor.Add(slotLength).Equal(start) {
				freeSlots = append(freeSlots, FreeSlot{
					Start: cursor,
					End:   cursor.Add(slotLength),
				})
				cursor = cursor.Add(slotLength)
			}
		}
		if end, _ := time.Parse(time.RFC3339, b.End); cursor.Before(end) {
			cursor = end
		}
	}

	// Добавим свободное время после последнего события
	for cursor.Add(slotLength).Before(to) || cursor.Add(slotLength).Equal(to) {
		freeSlots = append(freeSlots, FreeSlot{
			Start: cursor,
			End:   cursor.Add(slotLength),
		})
		cursor = cursor.Add(slotLength)
	}

	return freeSlots, nil
}

func NearestEvents(srv *calendar.Service, calendarID string) {
	// Получаем ближайшие события
	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(calendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(t).
		MaxResults(10).
		OrderBy("startTime").
		Do()
	if err != nil {
		log.Fatalf("Ошибка при получении событий: %v", err)
	}

	fmt.Println("Ближайшие события:")
	if len(events.Items) == 0 {
		fmt.Println("Нет событий.")
	} else {
		for _, item := range events.Items {
			start := item.Start.DateTime
			if start == "" {
				start = item.Start.Date
			}
			fmt.Printf("%s (%s)\n", item.Summary, start)
		}
	}

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
		fmt.Printf("- %s до %s\n", slot.Start.Format("15:04"), slot.End.Format("15:04"))
	}

	//calendar.NearestEvents(srv, calendarID)
}
