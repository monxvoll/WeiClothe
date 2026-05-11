# Behaviour view

This documentations is focused on the runtime behaviour of the system, how processes comunicate, concurrency, sychronization, performance an so on.

## Artefacts 

### Sequence Diagrams

#### User Register Module

```mermaid
sequenceDiagram
    autonumber
    participant Client as HTTP Client
    participant Go as Server GO
    participant KC as Keycloak
    participant DB as DB / Aurora

    Note over Client, DB: (Sign-Up)
    
    Client->>Go: POST /register (payload)
    Go->>KC: Create user (user, password)
    KC-->>Go: Return payload (includes generated UID)
    Go->>DB: INSERT (Metadata and UID)
    DB-->>Go: Save success
    Go-->>Client: Response OK (201 Created)
```

#### User Login Module

```mermaid
sequenceDiagram
    participant Client as HTTP Client
    participant Go as Backend GO
    participant KC as Keycloak
    participant Kafka as Kafka
    participant DB as Aurora

    Client->>Go: Credentials
    Go->>KC: Auth Request
    KC-->>Go: Payload
    
    Go->>Go: Success Verification
    
    alt Verification Failed
        Go-->>Client: 401 Unauthorized
    else Verification Successful
        Go->>Kafka: Request Metadata
        Go->>DB: Request Metadata
        DB-->>Go: Metadata Response
        Go-->>Client: 200 OK
    end
```

#### JWT Session Verification Flow

```mermaid
sequenceDiagram
    participant Client as User / Client
    participant Go as Server GO
    participant KC as Keycloak

    Note over Client, KC: JWT Session Verification Flow

    Client->>Go: Request with JWT
    
    %% Flujo de introspección dibujado por el usuario
    Go->>KC: Validate JWT 
    KC-->>Go: Authorization Status
    
    %% Bloque condicional obligatorio en UML
    alt JWT is Valid
        Go-->>Client: 200 OK
    else JWT is Invalid/Expired
        Go-->>Client: 401 Unauthorized
    end
```

### Clothes image ingestion & registration

```mermaid
sequenceDiagram
    autonumber
    participant C as Angular / Client
    participant G as Go API
    participant KC as Keycloak
    participant DB as Postgres DB
    participant S3 as S3 / MinIO
    participant K as Kafka
    participant V as "vusion-ml (consumer + ML)"

    C->>G: POST /clothes multipart (image + garment_type + JWT)
    G->>KC: Validate JWT
    KC-->>G: Auth status

    alt JWT invalid
        G-->>C: 401 Unauthorized
    else JWT valid
        G->>DB: INSERT garment row (image_url empty initially)
        G->>S3: PutObject raw/{user}/{id}/original.ext (stage raw bytes)
        G->>K: Publish analysis request (garment_id, staging_key, user_id)
        G-->>C: 201 Created (garment record)

        K->>V: Deliver message (analysis topic)
        V->>DB: UPDATE status = processing
        V->>S3: GetObject(staging_key) raw bytes to memory
        V->>V: run_pipeline (segmentation + classification)
        V->>S3: PutObject garments/{user}/{id}/processed.png
        V->>DB: UPDATE image_url, classification, detections, status completed
        V->>S3: DeleteObject(staging_key)
        Note over C,G: Async - client polls GET clothes by id or list, no webhook to Go API
    end
```

#### Clothes Recomendation Module

```mermaid
sequenceDiagram
    autonumber
    participant C as Client Service (Angular)
    participant R as Repository Service (API)
    participant DP as Data Processing (Python/ML)

    C->>R: 1. InfoRequest (user's request)
    activate R
    Note right of R: Data extraction from Aurora
    R->>R: 2. prepareContext()
    R->>DP: 3. instruction data (data transmission to process)
    activate DP
    Note over DP: Model execution for identify/recommend
    DP-->>R: 4. cleanData (Structure Information)
    deactivate DP
    R-->>C: 5. Recommendation Response (JSON)
    deactivate R
    C->>C: 6. Render UI (Show users)
```

#### Repository Module

```mermaid
sequenceDiagram
    autonumber
    participant C as Client Services
    participant R as Repository Service
    participant A as AWS-Aurora

    C->>R: 1. consultData()
    activate R
    R->>R: 2. findFunction()
    R->>A: 3. SQL(Consult)
    activate A
    A->>A: 4. Internal Process
    A-->>R: 5. response
    deactivate A
    R->>R: 6. formatData()
    R-->>C: 7. data
    deactivate R
```

### Comunication Diagram

```mermaid
flowchart TD
    %% Node Definitions
    Client(["Client Application (Angular/HTTP)"])
    GoBackend(["Go Backend Service (API / Repository)"])
    KC(["Keycloak"])
    Aurora(["AWS Aurora DB"])
    Kafka(["Kafka Event Broker"])
    DP(["Data Processing (Docker/ML)"])

    %% Communication Paths (Stretched with ---> and stacked with <br>)
    Client -- "1. Auth (Login/Register)<br>2. JWT Session Requests<br>3. UI Data & InfoRequests" ---> GoBackend
    
    GoBackend -- "1.1 Create User<br>2.1 Introspect JWT" ---> KC
    
    GoBackend -- "1.2 Emit Metadata Events" ---> Kafka
    
    GoBackend -- "1.3 Insert Metadata & UID<br>3.1 Extract Context" ---> Aurora
    
    GoBackend -- "3.2 Transmit Context" ---> DP
    
    %% Return Paths (Shortened text to prevent horizontal crashing)
    KC -. "Auth Status" .-> GoBackend
    DP -. "Cleaned Data" .-> GoBackend
    Aurora -. "Query Responses" .-> GoBackend
    GoBackend -. "HTTP Responses<br>Recommendation JSON" .-> Client
```
### Interaction Overview

```mermaid
flowchart TD

  %% ── START ──
  START([Start])

  %% ── TOP-LEVEL DECISION ──
  REQ_TYPE{Request type?}
  START --> REQ_TYPE

  %% ══════════════════════════════
  %% ── AUTH BRANCH ──
  %% ══════════════════════════════
  REQ_TYPE -- Auth --> AUTH_FORK{Auth action?}

  AUTH_FORK -- Login --> LOGIN["Login module
  POST credentials → KC auth
  → Kafka + Aurora"]

  AUTH_FORK -- Register --> REGISTER["Register module
  POST /register
  → KC create user
  → Aurora INSERT"]

  LOGIN --> VERIFY{Verification result?}
  REGISTER --> VERIFY

  VERIFY -- Fail --> E401_AUTH([401 Unauthorized])
  VERIFY -- Success --> OK_AUTH([200 / 201 + JWT])

  %% ══════════════════════════════
  %% ── APP REQUEST BRANCH ──
  %% ══════════════════════════════
  REQ_TYPE -- App request --> JWT_MOD["JWT verification
  Request + JWT → KC introspect
  ← Auth status"]

  JWT_MOD --> JWT_VALID{JWT valid?}
  JWT_VALID -- Invalid/Expired --> E401_JWT([401 Unauthorized])
  JWT_VALID -- Valid --> SVC_FORK{Service request?}

  %% ── REPOSITORY ──
  SVC_FORK -- Repo --> REPO["Repository module
  consultData() → findFunction()
  SQL → Aurora → formatData()"]

  %% ── RECOMMENDATION ──
  SVC_FORK -- Recommend --> REC["Recommendation module
  InfoRequest → prepareContext()
  → ML Docker ← cleanData JSON"]

  %% ── GARMENT UPLOAD ──
  SVC_FORK -- Upload --> UPLOAD["Garment upload pipeline
  POST /upload (image + metadata + JWT)
  → 202 Accepted"]

  UPLOAD --> WORKER["Worker / Consumer
  Pulls from Kafka
  → ML inference
  → Aurora INSERT
  → S3 store image"]

  WORKER --> NOTIFY["Job complete
  Webhook → GO Backend
  → Client notified"]

  REPO --> OK_APP([200 OK + data JSON])
  REC  --> OK_APP
  NOTIFY --> OK_APP

  %% ══════════════════════════════
  %% ── SHARED INFRASTRUCTURE ──
  %% ══════════════════════════════
  subgraph INFRA ["Go backend service (API / Repository)"]
    KC["Keycloak
    Auth & JWT issuer"]
    AURORA["AWS Aurora
    Metadata + queries"]
    KAFKA["Kafka
    Event broker"]
    ML["ML Docker / Python
    Recommendation + Classification"]
    S3["AWS S3
    Image storage"]
  end

  LOGIN -.-> KC
  LOGIN -.-> KAFKA
  LOGIN -.-> AURORA
  REGISTER -.-> KC
  REGISTER -.-> AURORA
  JWT_MOD -.-> KC
  REPO -.-> AURORA
  REC -.-> ML
  UPLOAD -.-> KAFKA
  WORKER -.-> ML
  WORKER -.-> AURORA
  WORKER -.-> S3

  %% ── CLIENT ──
  OK_AUTH --> CLIENT([Angular client — render UI])
  OK_APP  --> CLIENT
```

### State Diagrams

#### Upload Job Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Idle

    Idle --> Uploading : Client POST /upload

    Uploading --> ValidatingJWT : Request received by GO
    ValidatingJWT --> Rejected : JWT invalid / expired
    ValidatingJWT --> Queued : JWT valid → publish to Kafka

    Rejected --> [*] : 401 Unauthorized

    Queued --> Processing : GO consumes Kafka event
    Processing --> MLInference : Send image + metadata to ML Service
    MLInference --> Failed : ML error / timeout
    MLInference --> Persisting : Classification result received

    Persisting --> StoringAurora : INSERT metadata + result
    Persisting --> StoringS3 : Upload image

    StoringAurora --> Completed : Save OK
    StoringS3 --> Completed : Upload OK

    Failed --> Queued : Retry (if attempts < max)
    Failed --> [*] : Max retries exceeded → Dead letter

    Completed --> [*] : 200 OK → Client notified
```
#### JWT Session
```mermaid
stateDiagram-v2
    [*] --> Unauthenticated

    Unauthenticated --> Authenticating : POST credentials (Login)

    Authenticating --> Rejected : KC auth failed
    Authenticating --> Active : KC returns valid payload

    Rejected --> Unauthenticated : 401 → retry allowed
    Rejected --> [*] : Max attempts exceeded → locked

    Active --> Introspecting : Request arrives with JWT
    Introspecting --> Active : KC confirms valid
    Introspecting --> Expired : Token expired
    Introspecting --> Revoked : Token invalidated by KC

    Expired --> Unauthenticated : Force re-login → 401
    Revoked --> Unauthenticated : Force re-login → 401

    Active --> Unauthenticated : User logout
    Active --> [*] : Session terminated
```

#### User Account

```mermaid
stateDiagram-v2
    [*] --> Unregistered

    Unregistered --> Registering : POST /register (payload)

    Registering --> CreatingInKeycloak : GO calls KC create user
    CreatingInKeycloak --> RegistrationFailed : KC error (duplicate / invalid)
    CreatingInKeycloak --> PersistingMetadata : KC returns UID

    RegistrationFailed --> Unregistered : 400 / 409 → client retries
    RegistrationFailed --> [*] : Abandoned

    PersistingMetadata --> Active : Aurora INSERT success → 201 Created

    Active --> Suspended : Admin action
    Active --> Deleted : User/Admin deletion request

    Suspended --> Active : Admin reactivation
    Suspended --> Deleted : Admin permanent removal

    Deleted --> [*] : Account purged from KC + Aurora
```

