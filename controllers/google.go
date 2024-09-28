package controllers

import (
	"context"
	"github.com/blkcor/go-oauth2/config"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"net/url"
)

func GoogleLogin(c *fiber.Ctx) error {
	url := config.AppConfig.GoogleLoginConfig.AuthCodeURL("randomstate")
	c.Status(fiber.StatusSeeOther)
	c.Redirect(url)
	return c.JSON(url)
}

func GoogleCallback(c *fiber.Ctx) error {
	state := c.Query("state")
	if state != "randomstate" {
		return c.Status(fiber.StatusBadRequest).SendString("States don't Match!!")
	}

	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Code is missing")
	}

	googleConfig := config.GoogleConfig()

	// 配置代理
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	if err != nil {
		log.Printf("Invalid proxy URL: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Invalid proxy configuration")
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	client := &http.Client{
		Transport: transport,
	}

	// 将自定义客户端添加到上下文中
	ctx := context.WithValue(c.Context(), oauth2.HTTPClient, client)

	// 使用自定义客户端进行令牌交换
	token, err := googleConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("Exchange error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Code-Token Exchange Failed: " + err.Error())
	}

	// 使用令牌获取用户信息
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		log.Printf("User Data Fetch error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("User Data Fetch Failed: " + err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("User Data Fetch failed with status: %d, body: %s", resp.StatusCode, string(body))
		return c.Status(fiber.StatusInternalServerError).SendString("User Data Fetch Failed")
	}

	userData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("JSON Parsing error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("JSON Parsing Failed: " + err.Error())
	}

	return c.SendString(string(userData))
}
