
## Use case

The central rectangular boundary represents the core system (the Go backend), which encapsulates the 8 main use cases. The diagram defines three Actors interacting with the system: the WeiUser (primary actor), alongside Python AI Service and Keycloak (secondary external systems). Furthermore, the <<include>> relationship indicates that a base use case obligatorily invokes the functionality of the included use case as part of its execution (e.g., saving a clothe always triggers the AI analysis)
```mermaid
flowchart LR
    %% Actores
    WeiUser((WeiUser))
    PythonAI((Python AI\nService))
    Keycloak((Keycloak))

    %% Límite del sistema
    subgraph System["WeiClothe System [GO]"]
        direction TB
        UC1([Delete Clothes])
        UC2([Update Clothes])
        UC3([Search Clothes])
        UC4([Save Clothes])
        UC5([Analize Clothes])
        UC6([Login])
        UC7([Register])
    end

    %% Relaciones del WeiUser
    WeiUser --- UC1
    WeiUser --- UC2
    WeiUser --- UC3
    WeiUser --- UC4
    WeiUser --- UC6
    WeiUser --- UC7

    %% Relación de Inclusión (Include)
    UC4 -. "<< include >>" .-> UC5

    %% Relaciones de los actores/sistemas de la derecha
    UC5 --- PythonAI
    UC6 --- Keycloak
    UC7 --- Keycloak
```
