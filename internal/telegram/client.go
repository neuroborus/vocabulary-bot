package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultAPIBaseURL = "https://api.telegram.org"

type Client struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
}

type ClientOptions struct {
	BotToken   string
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(options ClientOptions) *Client {
	if options.BaseURL == "" {
		options.BaseURL = defaultAPIBaseURL
	}
	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}

	return &Client{
		botToken:   strings.TrimSpace(options.BotToken),
		baseURL:    strings.TrimRight(options.BaseURL, "/"),
		httpClient: options.HTTPClient,
	}
}

func (c *Client) GetUpdates(ctx context.Context, offset int, timeoutSeconds int) ([]Update, error) {
	values := url.Values{}
	if offset > 0 {
		values.Set("offset", strconv.Itoa(offset))
	}
	if timeoutSeconds > 0 {
		values.Set("timeout", strconv.Itoa(timeoutSeconds))
	}
	values.Set("allowed_updates", `["message","callback_query"]`)

	endpoint := c.methodURL("getUpdates")
	if encoded := values.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response apiResponse[[]Update]
	if err := c.doJSON(req, &response); err != nil {
		return nil, err
	}
	if !response.OK {
		return nil, fmt.Errorf("telegram getUpdates failed: %s", response.Description)
	}

	return response.Result, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	return c.sendMessage(ctx, chatID, text, "", nil)
}

func (c *Client) SendHTMLMessage(ctx context.Context, chatID int64, text string) error {
	return c.sendMessage(ctx, chatID, text, parseModeHTML, nil)
}

func (c *Client) SendHTMLMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard InlineKeyboardMarkup) error {
	return c.sendMessage(ctx, chatID, text, parseModeHTML, &keyboard)
}

func (c *Client) sendMessage(ctx context.Context, chatID int64, text, parseMode string, keyboard *InlineKeyboardMarkup) error {
	values := url.Values{}
	values.Set("chat_id", strconv.FormatInt(chatID, 10))
	values.Set("text", text)
	if parseMode != "" {
		values.Set("parse_mode", parseMode)
	}
	if keyboard != nil {
		payload, err := json.Marshal(keyboard)
		if err != nil {
			return err
		}
		values.Set("reply_markup", string(payload))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL("sendMessage"), strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var response apiResponse[json.RawMessage]
	if err := c.doJSON(req, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("telegram sendMessage failed: %s", response.Description)
	}

	return nil
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string) error {
	values := url.Values{}
	values.Set("callback_query_id", callbackQueryID)
	if text != "" {
		values.Set("text", text)
	}
	values.Set("show_alert", "false")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL("answerCallbackQuery"), strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var response apiResponse[bool]
	if err := c.doJSON(req, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("telegram answerCallbackQuery failed: %s", response.Description)
	}

	return nil
}

func (c *Client) SendChatAction(ctx context.Context, chatID int64, action string) error {
	values := url.Values{}
	values.Set("chat_id", strconv.FormatInt(chatID, 10))
	values.Set("action", strings.TrimSpace(action))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL("sendChatAction"), strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var response apiResponse[bool]
	if err := c.doJSON(req, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("telegram sendChatAction failed: %s", response.Description)
	}

	return nil
}

func (c *Client) SendDocument(ctx context.Context, chatID int64, path string, caption string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return err
	}
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return err
		}
		if err := writer.WriteField("parse_mode", parseModeHTML); err != nil {
			return err
		}
	}

	part, err := writer.CreateFormFile("document", filepath.Base(path))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL("sendDocument"), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var response apiResponse[json.RawMessage]
	if err := c.doJSON(req, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("telegram sendDocument failed: %s", response.Description)
	}

	return nil
}

func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand) error {
	body, err := json.Marshal(struct {
		Commands []BotCommand `json:"commands"`
	}{
		Commands: commands,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.methodURL("setMyCommands"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	var response apiResponse[bool]
	if err := c.doJSON(req, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("telegram setMyCommands failed: %s", response.Description)
	}
	if !response.Result {
		return fmt.Errorf("telegram setMyCommands failed")
	}

	return nil
}

func (c *Client) methodURL(method string) string {
	return c.baseURL + "/bot" + c.botToken + "/" + method
}

func (c *Client) doJSON(req *http.Request, target any) error {
	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("telegram http error %s", response.Status)
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}

	return nil
}

type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      T      `json:"result"`
}
