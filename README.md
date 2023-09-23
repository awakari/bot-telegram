# Awakari Telegram Bot

## User Created Chat

User creates a (group) chat with bot and selects a single subscription to read.
Bot creates a new chat record in the DB with `ACTIVE` state with a certain expiration time (e.g. now + 1 day).
Expired `ACTIVE` state should be treated as `INACTIVE`.
Chat id is unique, hence no more than 1 subscription may be linked to the chat.
When chat is created, bot can open a new **reader**.
The reader has a timeout (e.g. 1 day) and then recreated in the loop.
When recreating, it's necessary to check whether a chat exist in DB and update the status expiration time.
If not, then the reader should not be recreated anymore.

## User Left Chat

User leaves a (group) chat with bot.
Bot deletes the corresponding chat record from the DB.

## Subscription Deleted

When bot recreates a reader it immediately fails to read, response status is `NOT_FOUND`.
Bot:
1. Deletes the corresponding chat record from the DB.
2. Notifies user that the subscription has been deleted.

## Bot Start

In the loop, find the next chat record with state `INACTIVE`, set the state to `ACTIVE` atomically and return.
Resume the reader for every result. Exit the loop when there are no more `INACTIVE` chats in the DB.

## Bot Terminates

Bot does the best effort to set the state `INACTIVE` for all chats being read in the runtime.
No luck? Then the incorrect `ACTIVE` state will expire after some time.
