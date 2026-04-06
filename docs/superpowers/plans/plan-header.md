# EventChain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a full-stack campus event prediction market + fair ticketing system on Hyperledger Fabric 2.5 with a Liquid Glass UI.

**Architecture:** Vue 3 frontend communicates via REST with an Express backend that uses the Fabric Gateway SDK to interact with 4 Go chaincodes (token, event, prediction, ticket) running on a 3-org Fabric network. CouchDB provides rich query support. Authentication flows through Fabric CA for certificate management and bcrypt+SQLite for password verification.

**Tech Stack:** Hyperledger Fabric 2.5 / Go 1.21+ / Node.js 18+ / Express / Vue 3 / Pinia / ECharts / Element Plus / Docker Compose / CouchDB / Liquid Glass CSS

---

## Task Dependency Order

```
Task 1: Project Scaffolding (.gitignore, README, scripts)
   │
Task 2: Prerequisites Check (Docker, Go, Node, Fabric binaries)
   │
Task 3: Fabric Network Configuration (Docker Compose, configtx, CA)
   │
Task 4: Network Management Scripts (network.sh, channel, deploy)
   │
   ├── Task 5: token-chaincode (Go)
   │      │
   │      ├── Task 6: event-chaincode (Go)
   │      │
   │      ├── Task 7: prediction-chaincode (Go, AMM)
   │      │      │
   │      │      └── Task 8: ticket-chaincode (Go, cross-chaincode)
   │      │
   ├── Task 9: Server Scaffolding (Express + Fabric Gateway)
   │      │
   │      ├─�� Task 10: Auth System (register/login + Fabric CA)
   │      │
   │      ├── Task 11: Event + Prediction Routes
   │      │
   │      └── Task 12: Ticket + User Routes
   │
   └── Task 13: Client Scaffolding + Liquid Glass CSS
          │
          ├── Task 14: Base Components (GlassCard, Navbar, etc.)
          │
          ├── Task 15: Pinia Stores + API Layer
          │
          ├── Task 16: Home Page
          │
          ├── Task 17: Event Detail Page (ECharts)
          │
          ��── Task 18: Ticket Hall + Profile + Admin Pages
          │
          └── Task 19: Login Page

Task 20: Seed Data + Integration Test
Task 21: Final Polish + Demo Prep
```

**Parallelism:** Tasks 5-8 (chaincodes) can be developed in parallel. Tasks 13-19 (frontend) can start after Task 9 (server scaffolding) is done since the frontend uses the API.

---
