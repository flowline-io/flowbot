# Chromium with remote-debugging CDP for chat_agent.browser.driver=playwright.
# This is not the Playwright Test runner; agents use flowbot browser_* tools over CDP.
FROM mcr.microsoft.com/playwright:v1.55.0-jammy

USER root
EXPOSE 9222

# Playwright caches Chromium under /ms-playwright; resolve the chrome binary at start.
CMD ["bash", "-lc", "exec \"$(find /ms-playwright -type f -path '*/chrome-linux/chrome' | head -n1)\" --headless=new --no-sandbox --disable-gpu --disable-dev-shm-usage --remote-debugging-address=0.0.0.0 --remote-debugging-port=9222 about:blank"]
