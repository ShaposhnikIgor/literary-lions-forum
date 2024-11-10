# 📚 Literary Lions Forum 🦁

Welcome to **Literary Lions Forum** — an inviting and engaging digital space for book lovers to connect, review, and discuss literature! Built with a Go backend and a beautifully styled HTML/CSS frontend, the Literary Lions Forum provides a warm, welcoming atmosphere for sharing insights, sparking conversations, and discovering new books.

---

## 🌟 Features at a Glance

- **🔐 User Authentication**: Register and securely log in to the forum.
- **📝 Post & Comment**: Share your thoughts and comment on others’ posts.
- **👍 Like/Dislike**: Show appreciation or offer constructive feedback on posts and comments.
- **📚 Category-Based Browsing**: Explore posts by categories for a streamlined experience.
- **🔍 Search Functionality**: Easily find posts or topics of interest.

The Literary Lions Forum creates a lively online community, connecting people through their shared love of books.

---

## 🚀 Getting Started

### Prerequisites

- **Go** (version 1.20 or later)
- **Docker** (optional, for containerized deployment)
- **SQLite** (for data storage)

---

## 🛠 Project Setup

To get the Literary Lions Forum running locally or in a Docker container, follow these simple steps:

### 📥 Local Setup

1. **Clone the Repository**:
   ```bash
   git clone https://gitea.koodsisu.fi/irynazaporozhets/literary-lions.git
   cd literary-lions
   ```
2. **Install Go Dependencies**:
    ```bash
    go mod download
    ```
3. **Run the Application**:
    ```bash
    go run main.go
    ```
4. **Open in Browser**:
    Navigate to http://localhost:8080

### 🐳 Docker Setup

1. **Build the Docker Image**:
    ```bash
    docker build -t literary-lions .
    ```
2. **Run the Docker Container**:
    ```bash
    docker build -t literary-lions .
    ```
3. **Build the Docker Image**:
    ```bash
    docker run -p 8080:8080 literary-lions
    ```
4. **Open in Browser**:
    Navigate to http://localhost:8080

# 📚 Happy reading and connecting at the Literary Lions Forum! 🦁
