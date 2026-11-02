//go:build shiftreturneval

package shiftreturn

// Exchange schema including parent-approved pre-quality errata.
const schemaJSON = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "urn:0c-ime:shift-return-v2",
  "oneOf": [
    {
      "$ref": "#/$defs/fixture"
    },
    {
      "$ref": "#/$defs/query"
    },
    {
      "$ref": "#/$defs/candidate"
    },
    {
      "$ref": "#/$defs/snapshot"
    },
    {
      "$ref": "#/$defs/model"
    },
    {
      "$ref": "#/$defs/assignment"
    },
    {
      "$ref": "#/$defs/lock"
    },
    {
      "$ref": "#/$defs/row"
    },
    {
      "$ref": "#/$defs/report"
    },
    {
      "$ref": "#/$defs/sources"
    },
    {
      "$ref": "#/$defs/training_observation"
    },
    {
      "$ref": "#/$defs/candidate_occurrence"
    },
    {
      "$ref": "#/$defs/bank_task"
    },
    {
      "$ref": "#/$defs/generator_diagnostic"
    },
    {
      "$ref": "#/$defs/selected_model_manifest"
    }
  ],
  "$defs": {
    "segment": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "kind": {
          "enum": [
            "literal",
            "chinese",
            "pending"
          ]
        },
        "begin": {
          "type": "integer",
          "minimum": 0
        },
        "end": {
          "type": "integer",
          "minimum": 0
        },
        "reading": {
          "type": "string"
        }
      },
      "required": [
        "kind",
        "begin",
        "end",
        "reading"
      ]
    },
    "event": {
      "oneOf": [
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "key"
            },
            "key": {
              "type": "string",
              "pattern": "^[ -~]$"
            }
          },
          "required": [
            "type",
            "key"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "shift"
            }
          },
          "required": [
            "type"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "enter"
            }
          },
          "required": [
            "type"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "commit"
            }
          },
          "required": [
            "type"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "cancel"
            }
          },
          "required": [
            "type"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "backspace"
            }
          },
          "required": [
            "type"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "focus"
            }
          },
          "required": [
            "type"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "cursor"
            },
            "to": {
              "type": "integer",
              "minimum": 0
            }
          },
          "required": [
            "type",
            "to"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "paste"
            },
            "text": {
              "type": "string"
            }
          },
          "required": [
            "type",
            "text"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "mode"
            },
            "on": {
              "type": "boolean"
            }
          },
          "required": [
            "type",
            "on"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "caps"
            },
            "on": {
              "type": "boolean"
            }
          },
          "required": [
            "type",
            "on"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "select_literal"
            }
          },
          "required": [
            "type"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "select_candidate"
            },
            "candidate_id": {
              "type": "string"
            }
          },
          "required": [
            "type",
            "candidate_id"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "protect"
            },
            "on": {
              "type": "boolean"
            }
          },
          "required": [
            "type",
            "on"
          ]
        },
        {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "type": {
              "const": "pause"
            },
            "on": {
              "type": "boolean"
            }
          },
          "required": [
            "type",
            "on"
          ]
        }
      ]
    },
    "gold": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "segments": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/segment"
          },
          "minItems": 1
        },
        "accepted_surfaces": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "minItems": 1
        }
      },
      "required": [
        "segments",
        "accepted_surfaces"
      ]
    },
    "fixture": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "fixture"
        },
        "demo": {
          "type": "boolean"
        },
        "id": {
          "type": "string"
        },
        "prototype_id": {
          "type": "string"
        },
        "intent": {
          "enum": [
            "return",
            "literal",
            "compat"
          ]
        },
        "category": {
          "type": "string"
        },
        "stratum": {
          "type": "string"
        },
        "context_class": {
          "enum": [
            "none",
            "contrast",
            "identical_collision"
          ]
        },
        "source_ref": {
          "type": "string"
        },
        "source_priority": {
          "enum": [
            0,
            1
          ]
        },
        "families": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "minItems": 1
        },
        "prelude_events": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/event"
          }
        },
        "expected_prelude": {
          "type": "string"
        },
        "events": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/event"
          },
          "minItems": 1
        },
        "expected_raw": {
          "type": "string",
          "pattern": "^[ -~]*$"
        },
        "gold": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/gold"
          },
          "minItems": 1
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "id",
        "prototype_id",
        "intent",
        "category",
        "stratum",
        "context_class",
        "source_ref",
        "source_priority",
        "families",
        "prelude_events",
        "expected_prelude",
        "events",
        "expected_raw",
        "gold"
      ]
    },
    "query": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "query"
        },
        "demo": {
          "type": "boolean"
        },
        "raw": {
          "type": "string"
        },
        "eligible": {
          "type": "boolean"
        },
        "context_units": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "text": {
                "type": "string"
              },
              "reading": {
                "type": "string"
              },
              "literal": {
                "type": "boolean"
              },
              "original_keys": {
                "type": "string"
              }
            },
            "required": [
              "text",
              "reading",
              "literal",
              "original_keys"
            ]
          },
          "maxItems": 0
        },
        "custom_tokens": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "raw",
        "eligible",
        "context_units",
        "custom_tokens"
      ]
    },
    "candidate": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "candidate"
        },
        "demo": {
          "type": "boolean"
        },
        "id": {
          "type": "string"
        },
        "identity": {
          "type": "string"
        },
        "display_identity": {
          "type": "string"
        },
        "cut": {
          "type": "integer",
          "minimum": 0
        },
        "first_reading": {
          "type": "string"
        },
        "readings": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "parsed_end": {
          "type": "integer",
          "minimum": 0
        },
        "k": {
          "type": "integer",
          "minimum": 0
        },
        "tail_start": {
          "type": "integer",
          "minimum": 0
        },
        "segments": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/segment"
          }
        },
        "display_segments": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/segment"
          }
        },
        "display": {
          "type": "string"
        },
        "features": {
          "type": "array",
          "minItems": 13,
          "maxItems": 13,
          "items": {
            "type": "number"
          }
        },
        "kept": {
          "type": "boolean"
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "id",
        "identity",
        "display_identity",
        "cut",
        "first_reading",
        "readings",
        "parsed_end",
        "k",
        "tail_start",
        "segments",
        "display_segments",
        "display",
        "features",
        "kept"
      ]
    },
    "snapshot": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "snapshot"
        },
        "demo": {
          "type": "boolean"
        },
        "episode_id": {
          "type": "string"
        },
        "condition": {
          "type": "string"
        },
        "event_index": {
          "type": "integer",
          "minimum": 0
        },
        "key_offset": {
          "type": "integer",
          "minimum": 0
        },
        "app": {
          "type": "string"
        },
        "prelude": {
          "type": "string"
        },
        "display": {
          "type": "string"
        },
        "view": {
          "type": "string"
        },
        "commit": {
          "type": "string"
        },
        "handled": {
          "type": "boolean"
        },
        "eligible": {
          "type": "boolean"
        },
        "closed": {
          "type": "boolean"
        },
        "selected_id": {
          "type": [
            "string",
            "null"
          ]
        },
        "freeze": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "array",
              "items": {
                "$ref": "#/$defs/segment"
              }
            }
          ]
        },
        "generate_calls": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "score_calls": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "learning_updates": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "persistent_writes": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "protected": {
          "type": "boolean"
        },
        "resolution_count": {
          "type": "integer",
          "minimum": 0
        },
        "revision_count": {
          "type": "integer",
          "minimum": 0
        },
        "safety_status": {
          "enum": [
            "complete",
            "incomplete",
            "not_run"
          ]
        },
        "safety_reason": {
          "type": "string"
        },
        "query_id": {
          "type": [
            "string",
            "null"
          ]
        },
        "handoff": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "$ref": "#/$defs/handoff"
            }
          ]
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "episode_id",
        "condition",
        "event_index",
        "key_offset",
        "app",
        "prelude",
        "display",
        "view",
        "commit",
        "handled",
        "eligible",
        "closed",
        "selected_id",
        "freeze",
        "generate_calls",
        "score_calls",
        "learning_updates",
        "persistent_writes",
        "protected",
        "resolution_count",
        "revision_count",
        "safety_status",
        "safety_reason",
        "query_id",
        "handoff"
      ]
    },
    "model": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "model"
        },
        "demo": {
          "type": "boolean"
        },
        "name": {
          "enum": [
            "E0",
            "EL",
            "EW",
            "EC"
          ]
        },
        "feature_names": {
          "const": [
            "literal",
            "english_prefix_known",
            "prefix_has_digit",
            "prefix_has_punct",
            "prefix_numberlike",
            "first_prefix_log",
            "first_missing",
            "later_complete_syllables",
            "tail_bytes",
            "first_word_length",
            "multiword_coverage",
            "complete_mean_log",
            "complete_missing"
          ]
        },
        "active": {
          "type": "array",
          "items": {
            "type": "integer",
            "minimum": 0,
            "maximum": 12
          },
          "uniqueItems": true
        },
        "coefficients": {
          "type": "array",
          "minItems": 13,
          "maxItems": 13,
          "items": {
            "type": "number"
          }
        },
        "means": {
          "type": "array",
          "minItems": 13,
          "maxItems": 13,
          "items": {
            "type": "number"
          }
        },
        "stds": {
          "type": "array",
          "minItems": 13,
          "maxItems": 13,
          "items": {
            "type": "number"
          }
        },
        "counts": {
          "type": "array",
          "minItems": 13,
          "maxItems": 13,
          "items": {
            "type": "integer",
            "minimum": 0
          }
        },
        "margin": {
          "type": "number",
          "minimum": 0
        },
        "code_lock_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "fit_lock_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "training_loss": {
          "type": "number"
        },
        "gradient_l2": {
          "type": "number"
        },
        "selected_on": {
          "const": "dev"
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "name",
        "feature_names",
        "active",
        "coefficients",
        "means",
        "stds",
        "counts",
        "margin",
        "code_lock_sha256",
        "fit_lock_sha256",
        "training_loss",
        "gradient_l2",
        "selected_on"
      ]
    },
    "component": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "id": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "row_ids": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "minItems": 1
        },
        "split": {
          "enum": [
            "train",
            "dev",
            "test",
            "regression"
          ]
        },
        "primary_return": {
          "type": [
            "string",
            "null"
          ]
        },
        "primary_literal": {
          "type": [
            "string",
            "null"
          ]
        },
        "categories": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "primary_return_metadata": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "id": {
                  "type": "string"
                },
                "category": {
                  "type": "string"
                },
                "stratum": {
                  "type": "string"
                },
                "context_class": {
                  "enum": [
                    "none",
                    "contrast",
                    "identical_collision"
                  ]
                },
                "contrast_eligible": {
                  "type": "boolean"
                }
              },
              "required": [
                "id",
                "category",
                "stratum",
                "context_class",
                "contrast_eligible"
              ]
            }
          ]
        },
        "primary_literal_metadata": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "id": {
                  "type": "string"
                },
                "category": {
                  "type": "string"
                },
                "stratum": {
                  "type": "string"
                },
                "context_class": {
                  "enum": [
                    "none",
                    "contrast",
                    "identical_collision"
                  ]
                },
                "contrast_eligible": {
                  "type": "boolean"
                }
              },
              "required": [
                "id",
                "category",
                "stratum",
                "context_class",
                "contrast_eligible"
              ]
            }
          ]
        }
      },
      "required": [
        "id",
        "row_ids",
        "split",
        "primary_return",
        "primary_literal",
        "categories",
        "primary_return_metadata",
        "primary_literal_metadata"
      ]
    },
    "assignment": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "assignment"
        },
        "demo": {
          "type": "boolean"
        },
        "components": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/component"
          }
        },
        "fixtures_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "seen_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "components",
        "fixtures_sha256",
        "seen_sha256"
      ]
    },
    "lock": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "lock"
        },
        "demo": {
          "type": "boolean"
        },
        "baseline_commit": {
          "type": "string",
          "pattern": "^[0-9a-f]{40}$"
        },
        "source_commit": {
          "type": "string",
          "pattern": "^[0-9a-f]{40}$"
        },
        "dirty_diff_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "files": {
          "type": "object",
          "additionalProperties": {
            "type": "string",
            "pattern": "^[0-9a-f]{64}$"
          }
        },
        "commands": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "minItems": 1
        },
        "versions": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "go": {
              "type": "string"
            },
            "python": {
              "type": "string"
            },
            "numpy": {
              "type": "string"
            },
            "threads": {
              "const": 1
            }
          },
          "required": [
            "go",
            "python",
            "numpy",
            "threads"
          ]
        },
        "max_raw_bytes": {
          "const": 512
        },
        "max_kept_chinese": {
          "const": 256
        },
        "max_lex_queries": {
          "const": 250000
        },
        "max_decoder_calls": {
          "const": 100000
        },
        "max_pairs": {
          "const": 2000000
        },
        "max_artifact_bytes": {
          "const": 8589934592
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "baseline_commit",
        "source_commit",
        "dirty_diff_sha256",
        "files",
        "commands",
        "versions",
        "max_raw_bytes",
        "max_kept_chinese",
        "max_lex_queries",
        "max_decoder_calls",
        "max_pairs",
        "max_artifact_bytes"
      ]
    },
    "row": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "row"
        },
        "demo": {
          "type": "boolean"
        },
        "fixture_id": {
          "type": "string"
        },
        "component_id": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "condition": {
          "type": "string"
        },
        "intent": {
          "enum": [
            "return",
            "literal",
            "compat"
          ]
        },
        "primary": {
          "type": "boolean"
        },
        "support": {
          "enum": [
            "ok",
            "capacity_failure",
            "resource_exhausted",
            "unsupported"
          ]
        },
        "final_boundary_error": {
          "type": [
            "boolean",
            "null"
          ]
        },
        "final_surface_error": {
          "type": [
            "boolean",
            "null"
          ]
        },
        "final_english_error": {
          "type": [
            "boolean",
            "null"
          ]
        },
        "gold_survived": {
          "type": [
            "boolean",
            "null"
          ]
        },
        "first_correct": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "last_stable": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "temporary_error_lengths": {
          "type": "array",
          "items": {
            "type": "integer",
            "minimum": 0
          }
        },
        "revision_count": {
          "type": "integer",
          "minimum": 0
        },
        "resolution_count": {
          "type": "integer",
          "minimum": 0
        },
        "key_failures": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "commit_failures": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "app_rewrites": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "lock_violations": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "hypothetical_learning_updates": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "persistent_writes": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "protected_calls": {
          "type": [
            "integer",
            "null"
          ],
          "minimum": 0
        },
        "snapshots_path": {
          "type": "string"
        },
        "candidates_path": {
          "type": "string"
        },
        "primary_category": {
          "type": "string"
        },
        "context_class": {
          "enum": [
            "none",
            "contrast",
            "identical_collision"
          ]
        },
        "contrast_eligible": {
          "type": "boolean"
        },
        "safety_status": {
          "enum": [
            "complete",
            "incomplete",
            "not_run"
          ]
        },
        "safety_reason": {
          "type": "string"
        },
        "primary_metadata": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "id": {
                  "type": "string"
                },
                "category": {
                  "type": "string"
                },
                "stratum": {
                  "type": "string"
                },
                "context_class": {
                  "enum": [
                    "none",
                    "contrast",
                    "identical_collision"
                  ]
                },
                "contrast_eligible": {
                  "type": "boolean"
                }
              },
              "required": [
                "id",
                "category",
                "stratum",
                "context_class",
                "contrast_eligible"
              ]
            }
          ]
        },
        "generators_path": {
          "type": "string"
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "fixture_id",
        "component_id",
        "condition",
        "intent",
        "primary",
        "support",
        "final_boundary_error",
        "final_surface_error",
        "final_english_error",
        "gold_survived",
        "first_correct",
        "last_stable",
        "temporary_error_lengths",
        "revision_count",
        "resolution_count",
        "key_failures",
        "commit_failures",
        "app_rewrites",
        "lock_violations",
        "hypothetical_learning_updates",
        "persistent_writes",
        "protected_calls",
        "snapshots_path",
        "candidates_path",
        "primary_category",
        "context_class",
        "contrast_eligible",
        "safety_status",
        "safety_reason",
        "primary_metadata",
        "generators_path"
      ],
      "allOf": [
        {
          "if": {
            "properties": {
              "safety_status": {
                "const": "not_run"
              }
            }
          },
          "then": {
            "properties": {
              "key_failures": {
                "type": "null"
              },
              "commit_failures": {
                "type": "null"
              },
              "app_rewrites": {
                "type": "null"
              },
              "lock_violations": {
                "type": "null"
              },
              "hypothetical_learning_updates": {
                "type": "null"
              },
              "persistent_writes": {
                "type": "null"
              },
              "protected_calls": {
                "type": "null"
              }
            }
          }
        },
        {
          "if": {
            "properties": {
              "safety_status": {
                "const": "complete"
              }
            }
          },
          "then": {
            "properties": {
              "key_failures": {
                "type": "integer",
                "minimum": 0
              },
              "commit_failures": {
                "type": "integer",
                "minimum": 0
              },
              "app_rewrites": {
                "type": "integer",
                "minimum": 0
              },
              "lock_violations": {
                "type": "integer",
                "minimum": 0
              },
              "hypothetical_learning_updates": {
                "type": "integer",
                "minimum": 0
              },
              "persistent_writes": {
                "type": "integer",
                "minimum": 0
              },
              "protected_calls": {
                "type": "integer",
                "minimum": 0
              }
            }
          }
        }
      ]
    },
    "metric": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "errors": {
          "type": "integer",
          "minimum": 0
        },
        "n": {
          "type": "integer",
          "minimum": 0
        },
        "value": {
          "type": [
            "number",
            "null"
          ]
        },
        "wilson95": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "array",
              "minItems": 2,
              "maxItems": 2,
              "items": {
                "type": "number"
              }
            }
          ]
        },
        "independent_components": {
          "type": "boolean"
        },
        "status": {
          "enum": [
            "complete",
            "unavailable"
          ]
        },
        "observed_errors": {
          "type": "integer",
          "minimum": 0
        },
        "unavailable_components": {
          "type": "integer",
          "minimum": 0
        }
      },
      "required": [
        "errors",
        "n",
        "value",
        "wilson95",
        "independent_components",
        "status",
        "observed_errors",
        "unavailable_components"
      ],
      "allOf": [
        {
          "if": {
            "properties": {
              "status": {
                "const": "unavailable"
              }
            }
          },
          "then": {
            "properties": {
              "value": {
                "type": "null"
              },
              "wilson95": {
                "type": "null"
              }
            }
          }
        },
        {
          "if": {
            "properties": {
              "status": {
                "const": "complete"
              }
            }
          },
          "then": {
            "properties": {
              "n": {
                "type": "integer",
                "minimum": 1
              },
              "value": {
                "type": "number"
              },
              "wilson95": {
                "type": "array",
                "minItems": 2,
                "maxItems": 2,
                "items": {
                  "type": "number"
                }
              },
              "unavailable_components": {
                "const": 0
              }
            }
          }
        }
      ]
    },
    "gate": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "status": {
          "enum": [
            "pass",
            "fail",
            "unavailable"
          ]
        },
        "reason": {
          "type": "string"
        }
      },
      "required": [
        "status",
        "reason"
      ]
    },
    "report": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "report"
        },
        "demo": {
          "type": "boolean"
        },
        "outcome": {
          "enum": [
            "GO_OUTPUT_EXPERIMENT",
            "NO_GO_METHOD",
            "INCONCLUSIVE"
          ]
        },
        "primary_condition": {
          "const": "EC-revise"
        },
        "gates": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "G0": {
              "$ref": "#/$defs/gate"
            },
            "G1": {
              "$ref": "#/$defs/gate"
            },
            "G2": {
              "$ref": "#/$defs/gate"
            },
            "G3": {
              "$ref": "#/$defs/gate"
            },
            "G4": {
              "$ref": "#/$defs/gate"
            },
            "G5": {
              "$ref": "#/$defs/gate"
            }
          },
          "required": [
            "G0",
            "G1",
            "G2",
            "G3",
            "G4",
            "G5"
          ]
        },
        "conditions": {
          "type": "object",
          "additionalProperties": {
            "type": "object",
            "additionalProperties": {
              "$ref": "#/$defs/metric"
            }
          }
        },
        "paired": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "baseline": {
                "type": "string"
              },
              "challenger": {
                "type": "string"
              },
              "cohort": {
                "type": "string"
              },
              "n": {
                "type": "integer",
                "minimum": 0
              },
              "repaired": {
                "type": "integer",
                "minimum": 0
              },
              "worsened": {
                "type": "integer",
                "minimum": 0
              },
              "net": {
                "type": "integer"
              },
              "relative_error_reduction": {
                "type": [
                  "number",
                  "null"
                ]
              },
              "valid_bootstrap": {
                "type": "integer",
                "minimum": 0
              },
              "absolute_error_reduction_ci95": {
                "oneOf": [
                  {
                    "type": "null"
                  },
                  {
                    "type": "array",
                    "minItems": 2,
                    "maxItems": 2,
                    "items": {
                      "type": "number"
                    }
                  }
                ]
              },
              "discordant_components": {
                "type": "integer",
                "minimum": 0
              }
            },
            "required": [
              "baseline",
              "challenger",
              "cohort",
              "n",
              "repaired",
              "worsened",
              "net",
              "relative_error_reduction",
              "absolute_error_reduction_ci95",
              "valid_bootstrap",
              "discordant_components"
            ]
          }
        },
        "qualification": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "test_return_components": {
              "type": "integer",
              "minimum": 0
            },
            "test_literal_components": {
              "type": "integer",
              "minimum": 0
            },
            "test_english_components": {
              "type": "integer",
              "minimum": 0
            },
            "contrast_components": {
              "type": "integer",
              "minimum": 0
            },
            "category_components": {
              "type": "object",
              "additionalProperties": {
                "type": "integer",
                "minimum": 0
              }
            },
            "source_complete": {
              "type": "boolean"
            },
            "surface_gold_complete": {
              "type": "boolean"
            }
          },
          "required": [
            "test_return_components",
            "test_literal_components",
            "test_english_components",
            "contrast_components",
            "category_components",
            "source_complete",
            "surface_gold_complete"
          ]
        },
        "source_lock_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "models_sha256": {
          "type": "object",
          "additionalProperties": {
            "type": "string",
            "pattern": "^[0-9a-f]{64}$"
          }
        },
        "diagnostics_path": {
          "type": "string"
        },
        "safety": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "key_failures": {
              "type": [
                "integer",
                "null"
              ],
              "minimum": 0
            },
            "commit_failures": {
              "type": [
                "integer",
                "null"
              ],
              "minimum": 0
            },
            "app_rewrites": {
              "type": [
                "integer",
                "null"
              ],
              "minimum": 0
            },
            "lock_violations": {
              "type": [
                "integer",
                "null"
              ],
              "minimum": 0
            },
            "hypothetical_learning_updates": {
              "type": [
                "integer",
                "null"
              ],
              "minimum": 0
            },
            "persistent_writes": {
              "type": [
                "integer",
                "null"
              ],
              "minimum": 0
            },
            "protected_calls": {
              "type": [
                "integer",
                "null"
              ],
              "minimum": 0
            }
          },
          "required": [
            "key_failures",
            "commit_failures",
            "app_rewrites",
            "lock_violations",
            "hypothetical_learning_updates",
            "persistent_writes",
            "protected_calls"
          ]
        },
        "derived": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/derived"
          }
        },
        "safety_status": {
          "enum": [
            "complete",
            "incomplete",
            "not_run"
          ]
        },
        "safety_reason": {
          "type": "string"
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "outcome",
        "primary_condition",
        "gates",
        "conditions",
        "paired",
        "qualification",
        "source_lock_sha256",
        "models_sha256",
        "diagnostics_path",
        "safety",
        "derived",
        "safety_status",
        "safety_reason"
      ],
      "allOf": [
        {
          "if": {
            "properties": {
              "safety_status": {
                "const": "not_run"
              }
            }
          },
          "then": {
            "properties": {
              "safety": {
                "properties": {
                  "key_failures": {
                    "type": "null"
                  },
                  "commit_failures": {
                    "type": "null"
                  },
                  "app_rewrites": {
                    "type": "null"
                  },
                  "lock_violations": {
                    "type": "null"
                  },
                  "hypothetical_learning_updates": {
                    "type": "null"
                  },
                  "persistent_writes": {
                    "type": "null"
                  },
                  "protected_calls": {
                    "type": "null"
                  }
                }
              }
            }
          }
        },
        {
          "if": {
            "properties": {
              "safety_status": {
                "const": "complete"
              }
            }
          },
          "then": {
            "properties": {
              "safety": {
                "properties": {
                  "key_failures": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "commit_failures": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "app_rewrites": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "lock_violations": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "hypothetical_learning_updates": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "persistent_writes": {
                    "type": "integer",
                    "minimum": 0
                  },
                  "protected_calls": {
                    "type": "integer",
                    "minimum": 0
                  }
                }
              }
            }
          }
        }
      ]
    },
    "sources": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "sources"
        },
        "demo": {
          "type": "boolean"
        },
        "entries": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "id": {
                "type": "string"
              },
              "kind": {
                "enum": [
                  "synthetic-intent",
                  "public"
                ]
              },
              "reference": {
                "type": "string"
              },
              "version": {
                "type": "string"
              },
              "sha256": {
                "type": "string",
                "pattern": "^[0-9a-f]{64}$"
              },
              "license": {
                "type": "string"
              },
              "annotation_passes": {
                "const": 2
              },
              "adjudication_log": {
                "type": "string"
              },
              "annotator_ids": {
                "type": "array",
                "items": {
                  "type": "string"
                },
                "minItems": 2,
                "maxItems": 2,
                "uniqueItems": true
              },
              "cached_path": {
                "type": "string"
              }
            },
            "required": [
              "id",
              "kind",
              "reference",
              "version",
              "sha256",
              "license",
              "annotation_passes",
              "adjudication_log",
              "annotator_ids",
              "cached_path"
            ]
          }
        },
        "seen_prefixes": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "seen_contents": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "seen_raw": {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "entries",
        "seen_prefixes",
        "seen_contents",
        "seen_raw"
      ]
    },
    "derived": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "name": {
          "enum": [
            "interaction_I",
            "flow_gain",
            "context_extra_revision_gain"
          ]
        },
        "model": {
          "type": [
            "string",
            "null"
          ]
        },
        "cohort": {
          "type": "string"
        },
        "value": {
          "type": [
            "number",
            "null"
          ]
        },
        "bootstrap95": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "array",
              "minItems": 2,
              "maxItems": 2,
              "items": {
                "type": "number"
              }
            }
          ]
        },
        "valid_bootstrap": {
          "type": "integer",
          "minimum": 0
        }
      },
      "required": [
        "name",
        "model",
        "cohort",
        "value",
        "bootstrap95",
        "valid_bootstrap"
      ]
    },
    "candidate_occurrence": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "candidate_occurrence"
        },
        "demo": {
          "type": "boolean"
        },
        "fixture_id": {
          "type": "string"
        },
        "component_id": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "condition": {
          "enum": [
            "shared",
            "E0-fixed",
            "E0-revise",
            "EL-fixed",
            "EL-revise",
            "EW-fixed",
            "EW-revise",
            "EC-fixed",
            "EC-revise"
          ]
        },
        "event_index": {
          "type": "integer",
          "minimum": 0
        },
        "key_offset": {
          "type": "integer",
          "minimum": 0
        },
        "raw": {
          "type": "string"
        },
        "query_id": {
          "type": "string"
        },
        "candidate": {
          "$ref": "#/$defs/candidate"
        },
        "score": {
          "type": [
            "number",
            "null"
          ]
        },
        "allowed": {
          "type": "boolean"
        },
        "exclusion_reason": {
          "enum": [
            "none",
            "freeze"
          ]
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "fixture_id",
        "component_id",
        "condition",
        "event_index",
        "key_offset",
        "raw",
        "query_id",
        "candidate",
        "score",
        "allowed",
        "exclusion_reason"
      ]
    },
    "bank_task": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "bank_task"
        },
        "demo": {
          "type": "boolean"
        },
        "id": {
          "type": "string"
        },
        "prototype_id": {
          "type": "string"
        },
        "variant_index": {
          "type": "integer",
          "minimum": 0,
          "maximum": 4
        },
        "source_ref": {
          "type": "string"
        },
        "source_priority": {
          "enum": [
            0,
            1
          ]
        },
        "prompt": {
          "type": "string"
        },
        "intent": {
          "enum": [
            "return",
            "literal"
          ]
        },
        "category": {
          "type": "string"
        },
        "stratum": {
          "type": "string"
        },
        "context_class": {
          "enum": [
            "none",
            "contrast",
            "identical_collision"
          ]
        },
        "raw": {
          "type": "string"
        },
        "gold": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/gold"
          }
        },
        "families": {
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "prelude_events": {
          "type": "array",
          "items": {
            "$ref": "#/$defs/event"
          }
        },
        "expected_prelude": {
          "type": "string"
        },
        "designated_typo_spans": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "begin": {
                "type": "integer",
                "minimum": 0
              },
              "end": {
                "type": "integer",
                "minimum": 0
              }
            },
            "required": [
              "begin",
              "end"
            ]
          }
        },
        "source_edits": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "begin": {
                "type": "integer",
                "minimum": 0
              },
              "end": {
                "type": "integer",
                "minimum": 0
              },
              "replacement": {
                "type": "string"
              }
            },
            "required": [
              "begin",
              "end",
              "replacement"
            ]
          }
        },
        "annotations": {
          "type": "array",
          "minItems": 2,
          "maxItems": 2,
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "annotator_id": {
                "type": "string"
              },
              "intent": {
                "enum": [
                  "return",
                  "literal"
                ]
              },
              "reading_verified": {
                "type": "boolean"
              },
              "accepted": {
                "type": "boolean"
              },
              "labels_sha256": {
                "type": "string",
                "pattern": "^[0-9a-f]{64}$"
              },
              "adjudication_ref": {
                "type": "string"
              }
            },
            "required": [
              "annotator_id",
              "intent",
              "reading_verified",
              "accepted",
              "labels_sha256",
              "adjudication_ref"
            ]
          }
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "id",
        "prototype_id",
        "variant_index",
        "source_ref",
        "source_priority",
        "prompt",
        "intent",
        "category",
        "stratum",
        "context_class",
        "raw",
        "gold",
        "families",
        "prelude_events",
        "expected_prelude",
        "designated_typo_spans",
        "source_edits",
        "annotations"
      ]
    },
    "training_observation": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "training_observation"
        },
        "demo": {
          "type": "boolean"
        },
        "fixture_id": {
          "type": "string"
        },
        "component_id": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "event_index": {
          "type": "integer",
          "minimum": 0
        },
        "key_offset": {
          "type": "integer",
          "minimum": 0
        },
        "raw": {
          "type": "string"
        },
        "candidate": {
          "$ref": "#/$defs/candidate"
        },
        "gold_ids": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "uniqueItems": true
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "fixture_id",
        "component_id",
        "event_index",
        "key_offset",
        "raw",
        "candidate",
        "gold_ids"
      ]
    },
    "handoff": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "event": {
          "$ref": "#/$defs/event"
        },
        "action": {
          "enum": [
            "delegate",
            "commit_only",
            "restore_prelude"
          ]
        },
        "target_mode": {
          "enum": [
            "zhuyin",
            "english",
            "retain"
          ]
        },
        "prelude": {
          "type": "string"
        }
      },
      "required": [
        "event",
        "action",
        "target_mode",
        "prelude"
      ]
    },
    "generator_diagnostic": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "generator_diagnostic"
        },
        "demo": {
          "type": "boolean"
        },
        "fixture_id": {
          "type": "string"
        },
        "component_id": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "condition": {
          "enum": [
            "shared",
            "E0-fixed",
            "E0-revise",
            "EL-fixed",
            "EL-revise",
            "EW-fixed",
            "EW-revise",
            "EC-fixed",
            "EC-revise"
          ]
        },
        "event_index": {
          "type": "integer",
          "minimum": 0
        },
        "key_offset": {
          "type": "integer",
          "minimum": 0
        },
        "raw": {
          "type": "string"
        },
        "query_id": {
          "type": "string"
        },
        "generation_performed": {
          "type": "boolean"
        },
        "uncapped_ids": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "array",
              "items": {
                "type": "string",
                "pattern": "^h:[0-9a-f]{64}$"
              }
            }
          ]
        },
        "kept_ids": {
          "type": "array",
          "items": {
            "type": "string",
            "pattern": "^h:[0-9a-f]{64}$"
          }
        },
        "uncapped_count": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "integer",
              "minimum": 0
            }
          ]
        },
        "kept_count": {
          "type": "integer",
          "minimum": 0
        },
        "literal_id": {
          "type": "string",
          "pattern": "^h:[0-9a-f]{64}$"
        },
        "cap_pruned": {
          "type": [
            "boolean",
            "null"
          ]
        },
        "raw_overflow": {
          "type": "boolean"
        },
        "status": {
          "enum": [
            "ok",
            "capacity_failure",
            "resource_exhausted",
            "unsupported"
          ]
        },
        "reason": {
          "type": "string"
        },
        "limit_hit": {
          "enum": [
            "none",
            "raw_bytes",
            "probe_lex_queries",
            "best_calls"
          ]
        },
        "generate_calls": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "integer",
              "minimum": 0
            }
          ]
        },
        "probe_lex_queries": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "integer",
              "minimum": 0
            }
          ]
        },
        "best_calls": {
          "oneOf": [
            {
              "type": "null"
            },
            {
              "type": "integer",
              "minimum": 0
            }
          ]
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "fixture_id",
        "component_id",
        "condition",
        "event_index",
        "key_offset",
        "raw",
        "query_id",
        "generation_performed",
        "uncapped_ids",
        "kept_ids",
        "uncapped_count",
        "kept_count",
        "literal_id",
        "cap_pruned",
        "raw_overflow",
        "status",
        "reason",
        "limit_hit",
        "generate_calls",
        "probe_lex_queries",
        "best_calls"
      ],
      "allOf": [
        {
          "if": {
            "properties": {
              "status": {
                "const": "ok"
              }
            }
          },
          "then": {
            "properties": {
              "uncapped_ids": {
                "type": "array",
                "items": {
                  "type": "string",
                  "pattern": "^h:[0-9a-f]{64}$"
                }
              },
              "uncapped_count": {
                "type": "integer",
                "minimum": 0
              },
              "cap_pruned": {
                "type": "boolean"
              },
              "generate_calls": {
                "type": "integer",
                "minimum": 0
              },
              "probe_lex_queries": {
                "type": "integer",
                "minimum": 0
              },
              "best_calls": {
                "type": "integer",
                "minimum": 0
              }
            }
          }
        }
      ]
    },
    "selected_model_manifest": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "version": {
          "const": "shift-return-v2"
        },
        "kind": {
          "const": "selected_model_manifest"
        },
        "demo": {
          "type": "boolean"
        },
        "source_lock_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "fit_lock_sha256": {
          "type": "string",
          "pattern": "^[0-9a-f]{64}$"
        },
        "models": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "E0": {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "path": {
                  "type": "string",
                  "minLength": 1
                },
                "sha256": {
                  "type": "string",
                  "pattern": "^[0-9a-f]{64}$"
                }
              },
              "required": [
                "path",
                "sha256"
              ]
            },
            "EL": {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "path": {
                  "type": "string",
                  "minLength": 1
                },
                "sha256": {
                  "type": "string",
                  "pattern": "^[0-9a-f]{64}$"
                }
              },
              "required": [
                "path",
                "sha256"
              ]
            },
            "EW": {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "path": {
                  "type": "string",
                  "minLength": 1
                },
                "sha256": {
                  "type": "string",
                  "pattern": "^[0-9a-f]{64}$"
                }
              },
              "required": [
                "path",
                "sha256"
              ]
            },
            "EC": {
              "type": "object",
              "additionalProperties": false,
              "properties": {
                "path": {
                  "type": "string",
                  "minLength": 1
                },
                "sha256": {
                  "type": "string",
                  "pattern": "^[0-9a-f]{64}$"
                }
              },
              "required": [
                "path",
                "sha256"
              ]
            }
          },
          "required": [
            "E0",
            "EL",
            "EW",
            "EC"
          ]
        }
      },
      "required": [
        "version",
        "kind",
        "demo",
        "source_lock_sha256",
        "fit_lock_sha256",
        "models"
      ]
    }
  }
}
`
