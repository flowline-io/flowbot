# Chromium with remote-debugging CDP for chat_agent.browser.driver=playwright.
# This is not the Playwright Test runner; agents use flowbot browser_* tools over CDP.
FROM mcr.microsoft.com/playwright:v1.55.0-jammy

USER root

RUN apt-get update \
	&& apt-get install -y --no-install-recommends socat \
	&& rm -rf /var/lib/apt/lists/*

COPY deployments/playwright-cdp/entrypoint.sh /usr/local/bin/playwright-cdp-entrypoint
RUN chmod 0755 /usr/local/bin/playwright-cdp-entrypoint

EXPOSE 9222

# Chromium listens on 127.0.0.1:9223; entrypoint proxies 0.0.0.0:9222 via socat.
CMD ["playwright-cdp-entrypoint"]
