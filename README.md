# Awakari Telegram Bot

```shell
kubectl create secret generic bot-telegram \
  --from-literal=telegram=<TG_BOT_TOKEN> \
  --from-literal=support=<CHAT_ID_SUPPORT> \
  --from-literal=webhookToken=<WEBHOOK_TOKEN>
```
