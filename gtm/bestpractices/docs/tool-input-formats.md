# Tool Input Formats

Several GTM tools accept JSON as a string because Google models parameters as
recursive `type`/`key`/`value` objects. Use the shapes below instead of guessing
the wire format.

## Parameters

A scalar parameter has `type`, `key`, and `value` fields:

```json
[{"type":"template","key":"measurementId","value":"G-XXXXXXX"}]
```

Use `list` or `map` instead of `value` for nested parameters. When an update tool
says a JSON field can be omitted, omission preserves the existing value and `[]`
clears it.

## Tag sequencing

Setup and teardown tags are arrays referencing a tag name:

```json
[{"tagName":"12","stopOnSetupFailure":true}]
```

```json
[{"tagName":"34","stopTeardownOnFailure":false}]
```

## Trigger conditions

Conditions support `equals`, `contains`, `doesNotContain`, `startsWith`,
`endsWith`, and `matchRegex`. Each condition contains a parameter array with
`arg0` for the variable and `arg1` for the comparison value.

For `linkClick`, `click`, and `formSubmission`, Google silently drops
`autoEventFilter`; the server remaps those conditions to `filter`. A trigger
group uses this parameter shape:

```json
[{"key":"triggerIds","type":"list","list":[{"type":"triggerReference","value":"TRIGGER_ID"}]}]
```

## Transformations

Transformation types and their table fields are:

- `tf_allow_params`: `allowedParamsTable`, column `allowedParams`
- `tf_exclude_params`: `excludedParamsTable`, column `excludedParams`
- `tf_augment_event`: `augmentEventTable`, columns `paramName` and `paramValue`

The common fields are `matchingConditionsEnabled`, `allTagsExcept`,
`affectedTags`, `affectedTagTypes`, and `matchingConditionsTable`.

Example exclusion parameters:

```json
[
  {"key":"excludedParamsTable","type":"list","list":[{"type":"map","map":[{"key":"excludedParams","type":"template","value":"x-fb-ck-fbp"}]}]},
  {"key":"matchingConditionsEnabled","type":"boolean","value":"false"},
  {"key":"allTagsExcept","type":"boolean","value":"false"},
  {"key":"affectedTags","type":"list"},
  {"key":"affectedTagTypes","type":"list"}
]
```

## Custom templates

The template `name` field is an internal identifier. Change the visible name by
editing `displayName` inside the `___INFO___` section of `templateData`. Updating
a Gallery template can detach it from the Gallery source.
