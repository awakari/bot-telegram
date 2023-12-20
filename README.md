# Awakari Telegram Bot

```shell
kubectl create secret generic bot-telegram-tokens \
  --from-literal=telegram=<TELEGRAM_BOT_TOKEN> \
  --from-literal=payment=<PAYMENT_PROVIDER_TOKEN> \
  --from-literal=donation=<CHAT_ID_WITH_PINNED_INVOICE>
```

## Webhook Certificate

```shell
openssl req -newkey rsa:2048 -sha256 -nodes -keyout server.key -x509 -days 365 -out server.pem -subj "/O=Awakari/CN=tgbot.awakari.com"
```

```shell
kubectl create secret tls secret-bot-telegram-tls-server --key server.key --cert server.pem
```
