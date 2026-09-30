.PHONY: dev-backend dev-frontend

dev-backend:
	go run ./cmd/server

dev-frontend: frontend/node_modules/.package-lock.json
	@port_pid=$$(lsof -nP -tiTCP:5173 -sTCP:LISTEN 2>/dev/null | head -n 1); \
	if [ -n "$$port_pid" ]; then \
		frontend_dir=$$(cd frontend && pwd -P); \
		process_dir=$$(lsof -a -p "$$port_pid" -d cwd -Fn 2>/dev/null | sed -n 's/^n//p' | head -n 1); \
		if [ "$$process_dir" = "$$frontend_dir" ] && curl -fsS --max-time 2 http://127.0.0.1:5173/@vite/client >/dev/null 2>&1; then \
			printf 'InfoRadar 前端已运行：http://127.0.0.1:5173（PID %s）\n' "$$port_pid"; \
			exit 0; \
		fi; \
		printf '端口 5173 已被其他进程占用（PID %s），请先停止该进程。\n' "$$port_pid" >&2; \
		exit 1; \
	fi; \
	cd frontend && npm run dev

frontend/node_modules/.package-lock.json: frontend/package-lock.json
	cd frontend && npm ci
