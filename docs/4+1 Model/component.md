## component 

 The Go Core API acts as the primary entry point, exposing a REST interface to clients while securely managing identity delegation through Keycloak via OIDC. The architecture enforces a decoupling of concerns by utilizing Kafka as an event broker over TCP. This allows the Go backend to offload heavy image processing tasks asynchronously to the isolated Python AI Classifier service. Both core components persist data by interfacing directly with AWS Aurora (via TCP/SQL) and AWS S3 (via HTTPS/AWS SDK)

```mermaid
flowchart LR
   
    classDef comp fill:transparent,stroke:#333,stroke-width:2px;
    classDef ext fill:transparent,stroke:#666,stroke-width:2px,stroke-dasharray: 5 5;

    style CoreSystem fill:transparent,stroke:#333,stroke-width:2px
    style Infrastructure fill:transparent,stroke:#666,stroke-width:2px,stroke-dasharray: 5 5

    %% Actores/Clientes
    Client(["HTTP Client (Postman / Consumer)"]) 
    style Client fill:transparent,stroke:#333,stroke-width:2px

    %% Componentes Principales
    subgraph CoreSystem ["Core System"]
        GoAPI["«component» WeiClothe Core API (Go)"]
        PythonML["«component» AI Classifier Service (Python)"]
    end
    class GoAPI,PythonML comp;

    %% Infraestructura y Servicios Externos
    subgraph Infrastructure ["External Infrastructure"]
        KC["«component» Keycloak IAM"]
        Kafka["«component» Kafka Broker"]
        DB["«component» AWS Aurora"]
        S3["«component» AWS S3"]
    end
    class KC,Kafka,DB,S3 ext;

    %% Relaciones e Interfaces
    Client -- "REST API (HTTP)" --> GoAPI
    
    GoAPI -- "OIDC / REST" --> KC
    GoAPI -- "Produce Events (TCP)" --> Kafka
    GoAPI -- "SQL Query (TCP)" --> DB
    GoAPI -- "AWS SDK (HTTPS)" --> S3
    
    Kafka -- "Consume Events (TCP)" --> PythonML
    PythonML -- "SQL Query (TCP)" --> DB
    PythonML -- "AWS SDK (HTTPS)" --> S3
```