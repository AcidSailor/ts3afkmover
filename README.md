# ts3afkmover

Simple bot to move afk clients to the specified channel

## Environment Variables

| Name                    | Default                           | Required | Description                                                                                                          | Example                     |
|-------------------------|-----------------------------------|----------|----------------------------------------------------------------------------------------------------------------------|-----------------------------|
| TS3_URL                 | -                                 | True     | ts3 server url                                                                                                       | http://127.0.0.1:10080      |
| TS3_API_KEY             | -                                 | True     | ts3 server api key                                                                                                   | somekey                     |
| TS3_VSERVER_ID          | 1                                 | False    | Vserver Id                                                                                                           | 1                           |
| TS3_IDLE_TIME           | 60                                | False    | Idle time for client to be moved in minutes                                                                          | 60                          |
| TS3_IDLE_CHANNEL_ID     | -                                 | True     | Channel id for client to be moved into                                                                               | 10                          |
| TS3_IDLE_CHECK_INTERVAL | 5                                 | False    | Interval for checking idle condition in minutes                                                                      | 10                          |
| TS3_REQUEST_TIMEOUT     | 15                                | False    | Timeout for ts3 server requests in seconds                                                                           | 15                          |
| TS3_MESSAGE_TEMPLATE    | User %s was moved to Idle Channel | False    | Message to send when the user has been moved to idle channel, client username is passed as an argument automatically | This is some message for %s |