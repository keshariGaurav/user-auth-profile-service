# 🔨 ForgeIt

**ForgeIt** is a microservices-based platform designed to help developers showcase, discover, and monetize their coding projects. Whether you're an indie developer, a freelancer, or a startup, ForgeIt gives you a space to list your projects, collaborate, and build reputation in the dev community.

---

## 🧱 Features

* 🛠️ **Project Listings**
  List your code-based projects with descriptions, tags, and links. Each project is visible to the public and optimized for discovery.

* 💬 **Real-time Messaging**
  Connect with project owners, collaborators, or potential clients using WebSocket-based messaging.

* 🔐 **Authentication & Authorization**
  JWT-based user login and registration powered by a dedicated Go-based microservice using Fiber and MongoDB.

* 📧 **Email Notifications**
  Password reset, signup confirmation, and project updates sent via a standalone Go email microservice using Mailtrap (dev) and Amazon SES (prod).

* 💸 **Monetization**

  * ₹100/month per project listing
  * Optional ad placements for increased visibility

* 🧵 **Microservices Architecture**
  Built using multiple independent services for scalability and maintainability:

  * `auth-service` (Go + Fiber + MongoDB)
  * `email-service` (Go + RabbitMQ)
  * `chat-service` (WebSocket-based)
  * `project-service` (NestJS + PostgreSQL)
  * ...more coming soon

---

## 🧪 Tech Stack

| Layer      | Stack                                    |
| ---------- | ---------------------------------------- |
| Frontend   | React.js (planned)                       |
| Backend    | NestJS, Go (Fiber), WebSockets, RabbitMQ |
| Database   | PostgreSQL, MongoDB                      |
| Messaging  | RabbitMQ                                 |
| Email      | Mailtrap (dev), Amazon SES (prod)        |
| Auth       | JWT-based, refresh tokens (planned)      |
| Deployment | Docker + Kubernetes          |

---

## 🚧 Project Structure (WIP)

```
forgeit/
├── auth-service/           # Go + Fiber, MongoDB
├── email-service/          # Go + RabbitMQ
├── project-service/        # NestJS + PostgreSQL
├── chat-service/           # WebSocket server
├── gateway/                # (optional) API Gateway
└── frontend/               # React (planned)
```

---

## ⚙️ Getting Started

### Prerequisites

* Docker
* Go (v1.21+)
* Node.js (v18+)
* RabbitMQ
* MongoDB & PostgreSQL

### Clone the Repo

```bash
git clone https://github.com/yourusername/forgeit.git
cd forgeit
```

### Run with Docker (Example for dev)

```bash
docker-compose up --build
```

---

## 📬 Email Service Configuration

Set environment variables for the email service:

```env
MAIL_FROM=hello@forgeit.com
MAILTRAP_USERNAME=your_username
MAILTRAP_PASSWORD=your_password
```

---

## 📌 Roadmap

* [x] JWT-based auth
* [x] Password reset via email
* [x] Project listing with CRUD
* [ ] Admin dashboard
* [ ] Web UI for browsing projects
* [ ] Stripe/UPI integration
* [ ] GitHub API integration
* [ ] Project analytics dashboard

---

## 🤝 Contributing

Contributions, ideas, and pull requests are welcome!

---

## 📄 License

MIT License

---

## ✨ Credits

Built with 💻 and ☕ by [Gaurav Keshari](https://github.com/gauravkeshari)

---
