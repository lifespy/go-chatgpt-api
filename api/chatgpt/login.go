package chatgpt

import (
	"encoding/json"
	"github.com/tidwall/gjson"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/linweiyuan/go-chatgpt-api/api"

	http "github.com/bogdanfinn/fhttp"
)

func Login(c *api.LoginInfo) (*AuthResult, *Error) {

	userLogin := UserLogin{
		client: api.NewHttpClient(),
	}

	// get csrf token
	req, _ := http.NewRequest(http.MethodGet, csrfUrl, nil)
	req.Header.Set("User-Agent", api.UserAgent)
	resp, err := userLogin.client.Do(req)
	if err != nil {
		return nil, NewError(0, getCsrfTokenErrorMessage, err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			doc, _ := goquery.NewDocumentFromReader(resp.Body)
			alert := doc.Find(".message").Text()
			if alert != "" {
				return nil, NewError(resp.StatusCode, alert, err)
			}
		}
		return nil, NewError(resp.StatusCode, getCsrfTokenErrorMessage, err)
	}

	// get authorized url
	responseMap := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&responseMap)
	authorizedUrl, statusCode, err := userLogin.GetAuthorizedUrl(responseMap["csrfToken"])
	if err != nil {
		return nil, NewError(statusCode, getAuthorizedUrlErrorMessage, err)
	}

	// get state
	state, statusCode, err := userLogin.GetState(authorizedUrl)
	if err != nil {
		return nil, NewError(statusCode, getStateCodeErrorMessage, err)
	}

	// check username
	statusCode, err = userLogin.CheckUsername(state, c.Username)
	if err != nil {
		return nil, NewError(statusCode, getCheckUsernameErrorMessage, err)
	}

	// check password
	_, statusCode, err = userLogin.CheckPassword(state, c.Username, c.Password)
	if err != nil {
		return nil, NewError(statusCode, getCheckPasswordErrorMessage, err)
	}

	// get access token
	body, statusCode, err := userLogin.GetAccessToken("")
	if err != nil {
		return nil, NewError(statusCode, getAccessTokenErrorMessage, err)
	}

	if !gjson.Valid(body) {
		return nil, NewError(500, parseJsonErrorMessage, err)
	}
	parse := gjson.Parse(body)
	return &AuthResult{
		AccessToken:  parse.Get("accessToken").String(),
		RefreshToken: parse.Get("refresh_token").String(),
		PUID:         parse.Get("user.id").String(),
	}, nil
}

//goland:noinspection GoUnhandledErrorResult
func LoginApi(c *gin.Context) {
	var loginInfo api.LoginInfo
	if err := c.ShouldBindJSON(&loginInfo); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, api.ReturnMessage(api.ParseUserInfoErrorMessage))
		return
	}

	userLogin := UserLogin{
		client: api.NewHttpClient(),
	}

	// get csrf token
	req, _ := http.NewRequest(http.MethodGet, csrfUrl, nil)
	req.Header.Set("User-Agent", api.UserAgent)
	resp, err := userLogin.client.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.ReturnMessage(err.Error()))
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			doc, _ := goquery.NewDocumentFromReader(resp.Body)
			alert := doc.Find(".message").Text()
			if alert != "" {
				c.AbortWithStatusJSON(resp.StatusCode, api.ReturnMessage(strings.TrimSpace(alert)))
				return
			}
		}

		c.AbortWithStatusJSON(resp.StatusCode, api.ReturnMessage(getCsrfTokenErrorMessage))
		return
	}

	// get authorized url
	responseMap := make(map[string]string)
	json.NewDecoder(resp.Body).Decode(&responseMap)
	authorizedUrl, statusCode, err := userLogin.GetAuthorizedUrl(responseMap["csrfToken"])
	if err != nil {
		c.AbortWithStatusJSON(statusCode, api.ReturnMessage(err.Error()))
		return
	}

	// get state
	state, statusCode, err := userLogin.GetState(authorizedUrl)
	if err != nil {
		c.AbortWithStatusJSON(statusCode, api.ReturnMessage(err.Error()))
		return
	}

	// check username
	statusCode, err = userLogin.CheckUsername(state, loginInfo.Username)
	if err != nil {
		c.AbortWithStatusJSON(statusCode, api.ReturnMessage(err.Error()))
		return
	}

	// check password
	_, statusCode, err = userLogin.CheckPassword(state, loginInfo.Username, loginInfo.Password)
	if err != nil {
		c.AbortWithStatusJSON(statusCode, api.ReturnMessage(err.Error()))
		return
	}

	// get access token
	accessToken, statusCode, err := userLogin.GetAccessToken("")
	if err != nil {
		c.AbortWithStatusJSON(statusCode, api.ReturnMessage(err.Error()))
		return
	}

	c.Writer.WriteString(accessToken)
}
