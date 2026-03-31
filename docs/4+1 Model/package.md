
## package

The WeiClothe backend is built in Go using a standard domain-driven layout. The cmd/api package serves as the entry point, routing requests to the private internal/ modules where the core business logic (auth, inventory, and database connections) lives. We use pkg/ for reusable tools like our Kafka event publisher

```mermaid
flowchart TD
    classDef clean border:#333,stroke-width:2px,fill:none;
    classDef ext stroke:#666,stroke-width:2px,stroke-dasharray: 5 5,fill:none;

    %% BACKEND (Go Workspace)
   
    subgraph GoWorkspace ["Go Workspace (Backend API)"]
        direction TB
        Router["cmd/api (Entrypoint & Router)"]
        AuthB["internal/auth (Identity & JWT)"]
        InvB["internal/inventory (Business Logic)"]
        Events["pkg/events (Kafka Client)"]
        DB["internal/database (Aurora DB logic)"]
        Storage["internal/storage (S3 Uploads)"]
        
        Router -. "<<uses>>" .-> AuthB
        Router -. "<<uses>>" .-> InvB
        InvB -. "<<uses>>" .-> Events
        InvB -. "<<uses>>" .-> DB
        InvB -. "<<uses>>" .-> Storage
    end
    class GoWorkspace,Router,AuthB,InvB,Events,DB,Storage clean;

   
    %% PYTHON AI Workspace
   
    subgraph PythonAI ["Python Workspace (AI Service)"]
        direction TB
        Flask["api (Transport Layer)"]
        ML["core/ml_model (Inference)"]
        
        Flask -. "<<uses>>" .-> ML
    end
    class PythonAI,Flask,ML clean;

  
    %% INFRAESTRUCTURA EXTERNA
   
    KC["Keycloak (IAM)"]
    KafkaCluster["Kafka (Event Broker)"]
    AuroraDB["AWS Aurora (RDBMS)"]
    AWS_S3["AWS S3 (Object Storage)"]
    class KC,KafkaCluster,AuroraDB,AWS_S3 ext;

    
    %% COMUNICACIÓN DE RED
  
    AuthB ==>|HTTP/REST| KC
    Events ==>|TCP| KafkaCluster
    DB ==>|TCP/SQL| AuroraDB
    Storage ==>|HTTPS/API| AWS_S3
    
    %% Conexión Worker -> Python
    Events ==>|HTTP/REST| Flask
```