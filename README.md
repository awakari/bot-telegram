# Awakari Telegram Bot

```shell
kubectl create secret generic bot-telegram \
  --from-literal=telegram=<TGBOT_TOKEN> \
  --from-literal=payment=<PAYMENT_TOKEN> \
  --from-literal=donation=<DONATION_CHAT_ID> \
  --from-literal=loginCodeFromUserIds=<USER_ID_1>:<REPLICA_NUM_1>,<USER_ID_2>:<REPLICA_NUM_2>,...
```
