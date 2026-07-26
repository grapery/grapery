# Creation Agent Architecture

## Boundary

Natural-language creation flows enter `grapery-agent`.

Deterministic product operations stay on `grapery`:

- auth and user identity
- draft and published fragment records
- story, storyboard, and branch records
- generation tasks and polling state
- media assets, billing, moderation, and publishing

The iOS app can still call `grapery` directly for reads and explicit commands such as publish, save, delete, feed, upload, and detail hydration.

## Current Flow

Fragment creation now supports the unified agent entrypoint:

```text
iOS CreateFragmentChatViewModel
  -> grapery-agent /api/v1/agent/creation/sessions/:id/messages/stream
  -> grapery-agent generation service
  -> grapery /api/v1/fragments/generate
  -> grapery fragment generation task
  -> grapery-agent SSE progress/completed events
  -> iOS generation UI
```

The same protocol also accepts `targetType=story` and `targetType=branch`. Story messages are routed to the existing story generation runner. Branch messages are routed to the existing branch exploration runner and require `parentStoryboardId` in context.

## Creation Actions

The unified creation protocol should map natural language into stable business actions:

- `create`
- `continue`
- `revise_text`
- `append_image`
- `replace_image`
- `delete_image`
- `branch_from`
- `publish`
- `save_draft`

For fragments, `draftId` means continue or revise an unpublished fragment. `selectedImageIndex` is 1-based and means replace one image.

## API Contract

Create or reuse a session id, then send messages over SSE:

```http
POST /api/v1/agent/creation/sessions/:id/messages/stream
```

```json
{
  "message": "对方阵地的士兵看到这么厉害，就放弃阵地撤退了",
  "context": {
    "surface": "fragment_edit",
    "targetType": "fragment",
    "draftId": "frag_xxx"
  },
  "options": {
    "imageCount": 1,
    "branchCount": null,
    "sceneCount": null,
    "style": "fantasy",
    "visibility": "private",
    "aspectRatio": "9:16"
  }
}
```

SSE events use `creation.*` event types and keep the existing generation payload fields where possible: `taskId`, `draftFragmentId`, `progress`, `currentStep`, `messageKey`, `output`, and `tokensUsed`.
