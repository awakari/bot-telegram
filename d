[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:	export GIN_MODE=release
 - using code:	gin.SetMode(gin.ReleaseMode)

time=2025-05-16T17:43:33.114Z level=INFO msg="starting to listen the grpc API @ port #50051..."
[GIN-debug] GET    /v1/chat/:chatId          --> github.com/awakari/bot-telegram/service/chats.Handler.Confirm-fm (3 handlers)
[GIN-debug] POST   /v1/chat/:chatId          --> github.com/awakari/bot-telegram/service/chats.Handler.DeliverMessages-fm (3 handlers)
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8081
