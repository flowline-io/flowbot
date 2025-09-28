# Deployment Documentation

This directory contains deployment-related documentation and configuration files for FlowBot.

## File Descriptions

### `flowbot-agent.service`

Systemd service configuration file for running FlowBot Agent as a system service on Linux systems.

## Deployment Methods

### 1. Systemd Service Deployment (Recommended)

#### Installation Steps

1. Copy executable to system directory:

```bash
sudo cp flowbot /usr/local/bin/
sudo chmod +x /usr/local/bin/flowbot
```

2. Install service file:

```bash
sudo cp docs/deployment/flowbot-agent.service /etc/systemd/system/
```

3. Reload systemd configuration:

```bash
sudo systemctl daemon-reload
```

4. Enable and start service:

```bash
sudo systemctl enable flowbot-agent
sudo systemctl start flowbot-agent
```

#### Service Management

```bash
# Check service status
sudo systemctl status flowbot-agent

# Start service
sudo systemctl start flowbot-agent

# Stop service
sudo systemctl stop flowbot-agent

# Restart service
sudo systemctl restart flowbot-agent

# View logs
sudo journalctl -u flowbot-agent -f
```

### 2. Docker Deployment

Please refer to Docker-related files in the `deployments/` directory under the project root.

### 3. Manual Deployment

1. Ensure configuration files are properly set
2. Run executable directly:

```bash
./flowbot agent
```

## Deployment Checklist

- [ ] Configuration files are properly set
- [ ] Database connection is working
- [ ] Necessary environment variables are set
- [ ] Log directory has write permissions
- [ ] Network ports are accessible
- [ ] Service starts normally
- [ ] API interfaces respond correctly

## Troubleshooting

### Common Issues

1. **Service fails to start**

   - Check configuration file path and permissions
   - Check log output
   - Verify database connection

2. **Port conflicts**

   - Modify port settings in configuration file
   - Or use environment variable `FLOWBOT_SERVER_PORT`

3. **Permission issues**
   - Ensure service user has sufficient permissions
   - Check file and directory permissions
