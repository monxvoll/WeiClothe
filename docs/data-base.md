The following Entity-Relationship diagram reflects the current PostgreSQL schema (`GoClient/db/init.sql` plus migrations). It shows primary entities, attributes, relationships, and key constraints.

**Relationships**

| From | To | Cardinality | FK |
|------|-----|-------------|-----|
| `users` | `clothes` | 1:N | `clothes.user_id` → `users.sub_keycloak` |
| `users` | `user_style_preferences` | 1:1 | `user_style_preferences.user_id` → `users.sub_keycloak` |
| `clothes` | `styles` | 1:N | `styles.clothe_id` → `clothes.id` |
| `clothes` | `clothe_detections` | 1:N | `clothe_detections.clothe_id` → `clothes.id` (ON DELETE CASCADE) |

**Notable constraints**

- `clothes.status` ∈ `queued`, `processing`, `completed`, `failed`
- `clothes.source` ∈ `ai`, `manual`, `ai+manual`
- `clothes.confidence` ∈ [0, 1] when set
- `clothes.image_url` unique only when NOT NULL (partial index)
- `user_style_preferences.user_id` unique (one row per user)

```mermaid
classDiagram
    direction TB

    class USERS {
        +int id [PK]
        +string sub_keycloak [UK]
        +string first_name
        +string last_name
        +string nickname
        +string email [UK]
        +date date_birth
        +string gender
        +timestamp created_at
        +timestamp updated_at
    }

    class CLOTHES {
        +int id [PK]
        +string user_id [FK]
        +string image_url
        +int image_width
        +int image_height
        +string garment_type
        +string name
        +string classification_id
        +string category
        +string subcategory
        +string color
        +string pattern
        +string material
        +string season
        +string occasion
        +numeric confidence
        +string source
        +string model_name
        +string model_version
        +string status
        +string processing_error
        +timestamp processed_at
        +timestamp created_at
        +timestamp updated_at
    }

    class STYLES {
        +int id [PK]
        +int clothe_id [FK]
        +string style_id
        +string category
        +timestamp created_at
        +timestamp updated_at
    }

    class CLOTHE_DETECTIONS {
        +int id [PK]
        +int clothe_id [FK]
        +string class_name
        +numeric confidence
        +numeric bbox_x
        +numeric bbox_y
        +numeric bbox_w
        +numeric bbox_h
        +string mask_url
        +timestamp created_at
    }

    class USER_STYLE_PREFERENCES {
        +int id [PK]
        +string user_id [FK, UK]
        +text[] preferred_colors
        +text[] preferred_occasions
        +text[] preferred_seasons
        +text[] avoid_colors
        +timestamp created_at
        +timestamp updated_at
    }

    USERS "1" --> "0..*" CLOTHES : owns
    USERS "1" --> "0..1" USER_STYLE_PREFERENCES : prefers
    CLOTHES "1" --> "0..*" STYLES : tagged_with
    CLOTHES "1" --> "0..*" CLOTHE_DETECTIONS : detected_as
```

**Indexes** (non-PK)

| Table | Index |
|-------|--------|
| `clothes` | `user_id`, `status`, `category`, `processed_at`, `color`, `material`, `occasion`, `season`, `pattern`; partial unique on `image_url` |
| `clothe_detections` | `clothe_id` |
| `user_style_preferences` | `user_id` |
