Project: gym-crm-back

Language: Go

Architecture:
controller -> service -> repository

Rules:
- use context.Context everywhere
- use sqlx
- gin-gonic
- wrap errors
- write unit tests using testify