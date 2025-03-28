## 🚀 Getting Started

### Prerequisites
- Docker & Docker Compose
- Node.js 18+
- Go 1.20+
- Google Cloud account (for AI services)

### Installation

1. **Clone Repository**
```bash
git clone https://github.com/letrongminh/slideitin.git
cd slideitin
```

2. **Create Environment Files**
```bash
# API Service
cp backend/api/.env.example backend/api/.env
# Slides Generation Service
cp backend/slides-service/.env.example backend/slides-service/.env
```

3. **Configure Environment Variables**

Edit each .env file with your credentials:

```env: /backend/api/.env
# Required for Gemini AI

# Google Cloud Configuration (mostly overridden by docker-compose)
GOOGLE_CLOUD_PROJECT=xxx
SLIDES_SERVICE_URL=http://slides-service:8080
GCS_BUCKET_NAME=local-slideitin-files

# Server Configuration
PORT=8080

# CORS Configuration
FRONTEND_URL=http://localhost:3000

# Local Development Specific
FIRESTORE_EMULATOR_HOST=firestore-emulator:8080

```

```env: /backend/slides-service/.env
# Google Cloud Configuration (mostly overridden by docker-compose)
GEMINI_API_KEY=xxx
GOOGLE_CLOUD_PROJECT=xxx
GCS_BUCKET_NAME=local-slideitin-files

# Server Configuration
PORT=8080

# Local Development Specific
FIRESTORE_EMULATOR_HOST=firestore-emulator:8080

```

4. **Frontend Configuration**
```env:/Users/trongminhle/Downloads/MindShift_2025/Deal/HVKTQS/slideitin/frontend/.env.local
NEXT_PUBLIC_API_URL=http://localhost:8081/v1
```

Key notes:
- Get Gemini API key from [Google AI Studio](https://aistudio.google.com/)
- Encode service account JSON: `base64 -i service-account.json`
- Local Firestore emulator will auto-configure via docker-compose

To verify configurations:
```bash
# Check API service config
docker-compose run api env
```

## Tech Stack
### Frontend
- Next.js 14 
- TypeScript
- Tailwind CSS
```

### Docker Setup

1. **Start All Services**
```bash
# Build and start containers with logs
docker-compose up --build

# Detached mode (run in background)
docker-compose up -d
```

2. **Access Services**
- Frontend: http://localhost:3000
- API Docs: http://localhost:8081/docs
- Slides Service: http://localhost:8082/health
- Firestore Emulator: http://localhost:8080

3. **Verify Containers**
```bash
# List running containers
docker-compose ps

# Check API health
curl http://localhost:8081/health

# View service logs
docker-compose logs api
```

4. **Stop Services**
```bash
# Stop and remove containers
docker-compose down

# Remove volumes and networks
docker-compose down -v
```

### Configuration Tips
- Allocate at least 4GB RAM to Docker (AI services are resource-intensive)
- Add `.env` files as shown in previous sections
- First run may take 2-3 minutes for initial builds

### Development Commands
```bash
# Rebuild specific service
docker-compose build api

# Run single service
docker-compose run frontend npm run dev

# Access container shell
docker exec -it slideitin-api sh
```
