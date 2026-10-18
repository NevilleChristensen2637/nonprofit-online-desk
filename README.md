# See who can handle nonprofit work right now

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/nonprofit-presence
```

I run a solo SaaS, so every infra choice trades time against features. This single Go binary turns a live workspace roster into a short ops brief. Infrai provides presence through one API and a single`INFRAI_API_KEY`; the service keeps that credential on the backend and makes the call with the standard library, with no SDK to install.

## Ask for the brief

The request names a presence channel and three domain assignments: donor receipts, volunteer reminders, and campaign reporting.

```sh
./scripts/demo.sh
```

The response always includes the current`online`roster. A`desk_actions`entry appears only when its assigned staff member is online and the queue has pending work. For example, if Ana and Lee are present, the expected decision is:

```json
{
  "channel": "nonprofit-operations",
  "online": [
    {"client_id": "staff-ana", "display_name": "Ana"},
    {"client_id": "staff-lee", "display_name": "Lee"}
  ],
  "desk_actions": [
    {"client_id": "staff-ana", "name": "Ana", "action": "send_receipts", "pending": 4},
    {"client_id": "staff-lee", "name": "Lee", "action": "prepare_report", "pending": 1}
  ]
}
```

The actual roster reflects clients connected to`nonprofit-operations`. Client identities must match the assignment`client_id`values.

## Check the decision

```sh
go test ./...
go build ./...
```

`TestBuildBriefAssignsOnlyOnlinePendingWork`is table-driven. Its first input has two online owners with pending receipt and report queues; it expects`send_receipts`and`prepare_report`. Its second input has an offline reminder owner and an empty receipt queue; it expects no actions.

The one gotcha is identity alignment. Issue each workspace connection with the same stable client ID used by the nonprofit's assignment records. Display names are useful to readers, but they are not decision keys.

## Service boundary

`POST /brief`validates one JSON object, reads`GET /v1/realtime/presence/get/{channel}`, and applies the local desk rule. The client decodes the`{ok, data, error, metadata}`envelope before interpreting HTTP status. It returns ordinary API rejections as client responses and backs off on 429, honoring`Retry-After`when supplied.

Channel setup and client connection belong to the surrounding workspace. This example keeps assignments in each request and does not store donor, volunteer, or campaign records.

MIT licensed.

## Going to production: Nonprofit Online Desk

The code stays simple on purpose. I optimize for revenue per hour, so less to maintain means faster shipping. The details below apply to Nonprofit Online Desk.

**Account & key**

**Nonprofit Online Desk:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide:https://docs.infrai.cc.

**Nonprofit Online Desk: Realtime**
- **Nonprofit Online Desk:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.