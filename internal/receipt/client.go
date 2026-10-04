package receipt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	checkURL     = "https://proverkacheka.com/api/v1/check/get"
	pollDelay    = 2500 * time.Millisecond
	maxAttempts  = 6
	requestLimit = 15 * time.Second

	// code в ответе сервиса: 1 — успех, 2 и 4 — данные ещё не готовы, 3 — лимит токена
	codeOK        = 1
	codeNotReady  = 2
	codeLimit     = 3
	codeWaitRetry = 4
)

var (
	ErrNoToken     = errors.New("сканирование чеков не настроено на сервере")
	ErrCheckFailed = errors.New("не удалось получить данные чека")
	ErrRefund      = errors.New("возвраты и расходные чеки пока не поддерживаются")
)

type checker struct {
	tokens []string
	next   atomic.Uint32
	client *http.Client
}

func newChecker(tokens []string) *checker {
	return &checker{tokens: tokens, client: &http.Client{Timeout: requestLimit}}
}

func (c *checker) pickToken() string {
	i := c.next.Add(1) - 1
	return c.tokens[int(i)%len(c.tokens)]
}

type apiResponse struct {
	Code int `json:"code"`
	Data struct {
		JSON map[string]any `json:"json"`
	} `json:"data"`
}

// check запрашивает чек по строке из QR-кода. Пока сервис не отдал данные,
// опрашивает его; при исчерпании лимита переключается на следующий токен.
func (c *checker) check(ctx context.Context, qrraw string) (map[string]any, error) {
	if len(c.tokens) == 0 {
		return nil, ErrNoToken
	}

	limited := map[string]bool{}
	for attempt := 1; ; {
		token := c.pickToken()
		res, err := c.request(ctx, token, qrraw)
		if err != nil {
			return nil, err
		}

		switch {
		case res.Code == codeOK && res.Data.JSON != nil:
			return res.Data.JSON, nil
		case res.Code == codeLimit:
			limited[token] = true
			if len(limited) < len(c.tokens) {
				continue // другой токен, попытку ожидания не тратим
			}
		}

		if (res.Code == codeNotReady || res.Code == codeWaitRetry) && attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pollDelay):
			}
			attempt++
			continue
		}
		return nil, ErrCheckFailed
	}
}

func (c *checker) request(ctx context.Context, token, qrraw string) (apiResponse, error) {
	form := url.Values{"token": {token}, "qrraw": {qrraw}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, checkURL, strings.NewReader(form.Encode()))
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return apiResponse{}, ErrCheckFailed
	}
	defer resp.Body.Close()

	var out apiResponse
	dec := json.NewDecoder(resp.Body)
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return apiResponse{}, ErrCheckFailed
	}
	return out, nil
}

// --- Приведение ответа сервиса к нашей модели ---

// toReceipt берёт только нужные поля. Сервис местами отдаёт числа строками
// и наоборот, поэтому значения читаются через вспомогательные функции.
func toReceipt(j map[string]any) (Receipt, error) {
	if op := toInt(j["operationType"]); op != 0 && op != 1 {
		return Receipt{}, ErrRefund
	}

	date, ok := parseDate(j["ticketDate"])
	if !ok {
		date, ok = parseDate(j["dateTime"])
	}
	if !ok {
		return Receipt{}, ErrCheckFailed
	}

	r := Receipt{
		Key: fmt.Sprintf("%s-%s-%s",
			toString(j["fiscalDriveNumber"]), toString(j["fiscalDocumentNumber"]), toString(j["fiscalSign"])),
		Date:       date.Format("2006-01-02"),
		TotalSum:   toInt(j["totalSum"]),
		SellerINN:  toString(j["userInn"]),
		SellerName: firstNonEmpty(toString(j["user"]), toString(j["retailPlace"])),
	}
	if r.TotalSum <= 0 || r.Key == "--" {
		return Receipt{}, ErrCheckFailed
	}

	items, _ := j["items"].([]any)
	for _, raw := range items {
		m, _ := raw.(map[string]any)
		if m == nil {
			continue
		}
		r.Items = append(r.Items, Item{
			Name:     toString(m["name"]),
			Price:    toInt(m["price"]),
			Quantity: toFloat(m["quantity"]),
			Sum:      toInt(m["sum"]),
		})
	}
	return r, nil
}

func toString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func toFloat(v any) float64 {
	f, _ := strconv.ParseFloat(toString(v), 64)
	return f
}

func toInt(v any) int64 { return int64(math.Round(toFloat(v))) }

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseDate понимает форматы, в которых сервис отдаёт дату чека: компактный
// ФФД (20190202T1044), unix-секунды и обычные ISO-варианты.
func parseDate(v any) (time.Time, bool) {
	s := toString(v)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{"20060102T1504", "20060102T150405", time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	if sec, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(sec, 0), true
	}
	return time.Time{}, false
}
