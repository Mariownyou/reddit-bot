module github.com/mariownyou/reddit-bot

go 1.19

require github.com/vartanbeno/go-reddit/v2 v2.0.1

require github.com/mariownyou/go-reddit-uploader v0.0.0-20230601152731-c163d9920aa2 // indirect

require (
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/golang/protobuf v1.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/joho/godotenv v1.4.0
	golang.org/x/net v0.0.0-20190108225652-1e06a53dbb7e // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	google.golang.org/appengine v1.4.0 // indirect
)

// replace github.com/vartanbeno/go-reddit/v2 => ../go-reddit
