# Notification Configuration

This document describes the notification configuration templates supported by FlowBot.

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

## Usage in Configuration

Add notification URLs to your FlowBot configuration:

```yaml
notifications:
  channels:
    - name: "slack-alerts"
      url: "slack://YOUR_BOT_TOKEN/YOUR_TEAM_ID/YOUR_CHANNEL_ID"
    - name: "mobile-push"
      url: "pushover://YOUR_USER_KEY@YOUR_APP_TOKEN"
    - name: "internal-alerts"
      url: "message-pusher://USERNAME@HOSTNAME:PORT/CHANNEL/TOKEN"
```

## Testing Notifications

Use the FlowBot API to test notification configurations:

```bash
curl -X POST http://localhost:8080/notify/test \
  -H "Content-Type: application/json" \
  -d '{
    "channel": "slack-alerts",
    "message": "Test notification"
  }'
```
