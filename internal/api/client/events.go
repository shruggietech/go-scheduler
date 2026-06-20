package client

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/events"
)

// GetCalendar returns calendar occurrences in [from, to].
func (c *Client) GetCalendar(ctx context.Context, from, to time.Time) (server.CalendarResponse, error) {
	var out server.CalendarResponse
	path := "/v1/calendar?from=" + from.UTC().Format(time.RFC3339) + "&to=" + to.UTC().Format(time.RFC3339)
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

// StreamEvents opens the SSE event stream and invokes onEvent for each event
// until ctx is cancelled or the stream ends. It blocks; run it in a goroutine.
func (c *Client) StreamEvents(ctx context.Context, onEvent func(events.Event)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/events", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev events.Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err == nil {
			onEvent(ev)
		}
	}
}
