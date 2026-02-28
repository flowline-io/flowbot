# Notification Configuration

This document describes the notification providers supported by Flowbot.

Notification providers are implemented in `pkg/notify/`.

## Supported Providers

| Provider | Directory | Description |
|----------|-----------|-------------|
| Slack | `slack/` | Channel/user notifications |
| Pushover | `pushover/` | Mobile push notifications |
| ntfy | `ntfy/` | Self-hosted push notifications |
| Message Pusher | `message-pusher/` | Custom internal notifications |

## Slack

Send notifications to Slack channels or users.

### Configuration Format

```
slack://{tokenA}/{tokenB}/{tokenC}
```

### Parameters

- `tokenA` - Bot user OAuth token
- `tokenB` - Bot token (optional)
- `tokenC` - Channel or user ID

### Example

```
slack://YOUR_BOT_TOKEN_HERE/YOUR_TEAM_ID/YOUR_CHANNEL_ID
```

## Pushover

Send push notifications via Pushover service.

### Configuration Formats

```
pushover://{user_key}@{token}
pushover://{user_key}@{token}/{targets}
```

### Parameters

- `user_key` - Your Pushover user key
- `token` - Your application's API token
- `targets` - Specific device names (optional)

### Examples

```
# Send to all devices
pushover://u123abc@a456def

# Send to specific devices
pushover://u123abc@a456def/iphone,android
```

## ntfy

Send push notifications via [ntfy](https://ntfy.sh), a self-hosted push notification service.

### Configuration Format

```
ntfy://{server}/{topic}
```

### Parameters

- `server` - ntfy server URL (e.g., `ntfy.sh` or your self-hosted instance)
- `topic` - Target topic name

### Example

```
ntfy://ntfy.sh/my-alerts
```

## Message Pusher

Custom message pusher service for internal notifications.

### Configuration Formats

```
message-pusher://{user}@{domain}/{channel}/{token}
message-pusher://{user}@{host}:{port}/{channel}/{token}
```

### Parameters

- `user` - Username for authentication
- `domain` - Service domain name
- `host` - Service host address
- `port` - Service port number
- `channel` - Target channel or topic
- `token` - Authentication token

### Examples

```
# Using domain name
message-pusher://admin@example.com/general/abc123token

# Using host and port
message-pusher://admin@192.168.1.100:8080/alerts/xyz789token
```
