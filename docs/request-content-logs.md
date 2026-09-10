# Request content logs

New supported HTTP requests are recorded for all accounts by default. Administrators can view all records. In Users → row menu → Request content access, an administrator can grant or revoke a customer’s access to their own content. Customer access defaults to denied and is enforced by the API, including list previews. Revocation does not stop recording or delete records. `REQUEST_CONTENT_LOG_ENABLED=false` is an emergency global capture switch; it does not grant customer access. Communicate the recording policy to customers.

Supported routes: `/v1/chat/completions`, `/v1/responses`, `/v1/messages`, and Gemini `generateContent` / `streamGenerateContent`. WebSockets, files, image/video/audio endpoints and Playground are not captured. This is a record of the client's request and the response written back to it, not the provider-side transformed payload or a reconstructed Codex task.

The usage-log Details column shows the trailing user message excerpt when captured. Click it to view the request's text messages and response. Context/tool messages are collapsed. No AI intent inference or cross-request grouping is performed. A trailing tool result is labelled as a follow-up, not a new question. Attachments, URLs of attachments, headers and encrypted reasoning are omitted; tool calls are represented by markers rather than their full arguments.

Storage is in the **primary** database (`request_contents`), also when `LOG_SQL_DSN` points to ClickHouse. Content expires after 30 days and is excluded from reads immediately on expiry. A background job removes expired rows in batches of 500 each minute, with a 30-second time budget. Each direction stores up to 200 messages, 8,000 Unicode characters per message and 48 KiB of JSON; the response capture buffer is capped at 1 MiB. Exceeding these bounds is visibly marked as truncation. An interrupted stream or missing completion marker is not shown as complete. These limits do not truncate or alter the relayed response.

Details are loaded on demand through session-authenticated `/api/log/self/content/:request_id` (owner only) or `/api/log/content/:request_id` (administrator). Responses use `Cache-Control: no-store`. Bodies are not included in usage-log lists or CSV exports. Missing, expired and other-account records have the same empty response.

After enabling, send a test request, open its usage-log details, and verify the recorded question/model/answer and account isolation. Old requests cannot be recovered. Rollback: disable the environment flag and redeploy; the additive table can remain and retention continues. A successful release alone does not prove recording and customer access on a live request.

## Verification (2026-09-11)

- SQLite 3.50.4, MySQL 8.4.9, PostgreSQL 16.14: `TEST_MYSQL_DSN=... TEST_POSTGRES_DSN=... go test ./middleware -run '^TestRequestContent' -v -count=1`. Covers unchanged streamed responses, completion detection, bounded capture, fresh/additive migration twice, preservation of existing billing records, request ID uniqueness, owner isolation, default-denied customer access, grant/revoke, and expiry/deletion.
- Real application startup on all three engines: fresh startup twice; previous `release` (`14729d962`) startup followed by the new executable twice. All reached the listening state without migration errors.
- `GOWORK=off go vet ./...`, `GOWORK=off make test`, frontend typecheck, focused lint, frontend tests and production build.
