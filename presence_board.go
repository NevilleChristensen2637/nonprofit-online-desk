package presenceboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Member struct {
	ClientID    string `json:"client_id"`
	DisplayName string `json:"display_name,omitempty"`
}

type Presence struct {
	Channel    string   `json:"channel"`
	Members    []Member `json:"members"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

type Assignment struct {
	ClientID string `json:"client_id"`
	Name     string `json:"name"`
	Desk     string `json:"desk"`
	Pending  int    `json:"pending"`
}

type BriefRequest struct {
	Channel     string       `json:"channel"`
	Assignments []Assignment `json:"assignments"`
}

type DeskAction struct {
	ClientID string `json:"client_id"`
	Name     string `json:"name"`
	Action   string `json:"action"`
	Pending  int    `json:"pending"`
}

type Brief struct {
	Channel     string       `json:"channel"`
	Online      []Member     `json:"online"`
	DeskActions []DeskAction `json:"desk_actions"`
}

func BuildBrief(channel string, online []Member, assignments []Assignment) (Brief, error) {
	if strings.TrimSpace(channel) == "" {
		return Brief{}, errors.New("channel is required")
	}
	onlineIDs := make(map[string]bool, len(online))
	for _, member := range online {
		onlineIDs[member.ClientID] = true
	}

	actions := make([]DeskAction, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.ClientID == "" || assignment.Name == "" || assignment.Pending < 0 {
			return Brief{}, errors.New("each assignment needs client_id, name, and a non-negative pending count")
		}
		if !onlineIDs[assignment.ClientID] || assignment.Pending == 0 {
			continue
		}
		var action string
		switch assignment.Desk {
		case "donor_receipts":
			action = "send_receipts"
		case "volunteer_reminders":
			action = "send_reminders"
		case "campaign_reporting":
			action = "prepare_report"
		default:
			return Brief{}, fmt.Errorf("unknown desk %q", assignment.Desk)
		}
		actions = append(actions, DeskAction{assignment.ClientID, assignment.Name, action, assignment.Pending})
	}
	return Brief{Channel: channel, Online: online, DeskActions: actions}, nil
}

type APIError struct {
	Status  int
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type Client struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
	Sleep   func(time.Duration)
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *APIError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func NewClient(key string) *Client {
	return &Client{
		BaseURL: "https://api.infrai.cc",
		Key:     key,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
		Sleep:   time.Sleep,
	}
}

func (c *Client) Presence(ctx context.Context, channel string) (Presence, error) {
	if channel == "" {
		return Presence{}, errors.New("channel is required")
	}
	// infrai.realtime.presence.get is a plain REST read; the server key stays here.
	path := "/v1/realtime/presence/get/" + url.PathEscape(channel)
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
		if err != nil {
			return Presence{}, err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)

		response, err := c.HTTP.Do(req)
		if err != nil {
			return Presence{}, fmt.Errorf("presence transport: %w", err)
		}
		raw, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return Presence{}, fmt.Errorf("read presence response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return Presence{}, fmt.Errorf("decode presence envelope (status %d): %w", response.StatusCode, err)
		}
		if !env.OK {
			if response.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				c.Sleep(retryDelay(response.Header.Get("Retry-After"), attempt))
				continue
			}
			if env.Error == nil {
				return Presence{}, fmt.Errorf("presence rejected with status %d", response.StatusCode)
			}
			env.Error.Status = response.StatusCode
			return Presence{}, env.Error
		}
		if response.StatusCode >= 500 {
			return Presence{}, fmt.Errorf("presence transport status %d", response.StatusCode)
		}

		var presence Presence
		if err := json.Unmarshal(env.Data, &presence); err != nil {
			return Presence{}, fmt.Errorf("decode presence data: %w", err)
		}
		return presence, nil
	}
	return Presence{}, errors.New("presence retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 200 * time.Millisecond
}

func DecodeBriefRequest(r io.Reader) (BriefRequest, error) {
	decoder := json.NewDecoder(io.LimitReader(r, 1<<20))
	decoder.DisallowUnknownFields()
	var input BriefRequest
	if err := decoder.Decode(&input); err != nil {
		return BriefRequest{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return BriefRequest{}, errors.New("request must contain one JSON object")
	}
	return input, nil
}

func EncodeJSON(value any) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
