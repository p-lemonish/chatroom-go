package main

type Message struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Roomname string `json:"roomname"`
	Text     string `json:"text"`
}

type user struct {
	Username string
	Auth     string
}
