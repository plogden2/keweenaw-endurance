# Keweenaw Endurance Syndicate Race Timing System - Specification

## Project Overview

A comprehensive web application for managing endurance races in the Keweenaw area, featuring race indexing, live timing tracking, and race history tools. The system supports both time-based and lap-based race formats with RFID integration for participant tracking.

## Technology Stack

- **Frontend**: Vue.js 3 with Composition API
- **Backend**: Go (Golang) with Gin framework
- **Database**: PostgreSQL 14+
- **Containerization**: Docker and Docker Compose
- **Cloud Platform**: Google Cloud Platform (GCP)
- **RFID Hardware**: Proxmark3 RFID reader/writer

## System Architecture

### Container Services
1. **Frontend Service** (Vue.js application)
2. **Backend API Service** (Go REST API)
3. **PostgreSQL Database Service**
4. **Redis Cache Service** (for session management and caching)
5. **Nginx Reverse Proxy** (for routing and SSL termination)

### Google Cloud Infrastructure
- **Cloud Run** for containerized services
- **Cloud SQL** for managed PostgreSQL
- **Cloud Storage** for race photos and static assets
- **Cloud CDN** for content delivery
- **Cloud Monitoring** for observability

## Database Schema

### Core Tables

#### Events
```sql
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    event_date DATE NOT NULL,
    location VARCHAR(500),
    website_url VARCHAR(500),
    status VARCHAR(50) CHECK (status IN ('upcoming', 'active', 'completed', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Races
```sql
CREATE TABLE races (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    race_type VARCHAR(50) CHECK (race_type IN ('time_based', 'lap_based')),
    distance_km DECIMAL(10,2), -- for time-based races
    duration_minutes INTEGER, -- for lap-based races
    start_time TIMESTAMP,
    status VARCHAR(50) CHECK (status IN ('scheduled', 'active', 'finished', 'cancelled')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Participants
```sql
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID REFERENCES races(id) ON DELETE CASCADE,
    bib_number VARCHAR(20) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    gender VARCHAR(10) CHECK (gender IN ('male', 'female', 'other')),
    age INTEGER,
    rfid_tag_uid VARCHAR(100) UNIQUE,
    status VARCHAR(50) CHECK (status IN ('registered', 'started', 'finished', 'dnf', 'dns')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Timing Checkpoints
```sql
CREATE TABLE timing_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID REFERENCES races(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    checkpoint_type VARCHAR(50) CHECK (checkpoint_type IN ('start', 'finish', 'intermediate')),
    distance_from_start_km DECIMAL(10,2),
    location_description VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Timing Records
```sql
CREATE TABLE timing_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    participant_id UUID REFERENCES participants(id) ON DELETE CASCADE,
    checkpoint_id UUID REFERENCES timing_checkpoints(id) ON DELETE CASCADE,
    timestamp TIMESTAMP NOT NULL,
    local_timestamp TIMESTAMP NOT NULL, -- for offline functionality
    device_id VARCHAR(100), -- RFID device identifier
    sync_status VARCHAR(50) DEFAULT 'synced' CHECK (sync_status IN ('synced', 'pending_sync', 'failed_sync')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### Categories
```sql
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    race_id UUID REFERENCES races(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category_type VARCHAR(50) CHECK (category_type IN ('overall', 'male', 'female', 'age_group', 'custom')),
    age_min INTEGER,
    age_max INTEGER,
    gender_filter VARCHAR(10),
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## API Endpoints

### Event Management
```
GET    /api/events                    - List all events
GET    /api/events/:id               - Get event details
POST   /api/events                    - Create new event
PUT    /api/events/:id               - Update event
DELETE /api/events/:id               - Delete event
```

### Race Management
```
GET    /api/events/:eventId/races     - List races for event
GET    /api/races/:id                 - Get race details
POST   /api/events/:eventId/races     - Create race
PUT    /api/races/:id                 - Update race
DELETE /api/races/:id                 - Delete race
```

### Participant Management
```
GET    /api/races/:raceId/participants - List participants
GET    /api/participants/:id           - Get participant details
POST   /api/races/:raceId/participants - Register participant
PUT    /api/participants/:id           - Update participant
DELETE /api/participants/:id           - Remove participant
```

### Timing Data
```
GET    /api/races/:raceId/results      - Get race results
GET    /api/races/:raceId/leaderboard  - Get leaderboard data
GET    /api/timing/live/:raceId        - Get live timing data
POST   /api/timing/record              - Record timing event
PUT    /api/timing/records/:id         - Update timing record
```

### RFID Management
```
POST   /api/rfid/write-tag             - Write RFID tag
GET    /api/rfid/scan/:uid             - Lookup participant by RFID
POST   /api/rfid/manual-entry          - Manual timing entry
GET    /api/rfid/sync-status           - Get sync status
POST   /api/rfid/sync-pending          - Sync pending records
```

## Frontend Components

### Landing Page (`/`)
- Hero section with upcoming races highlight reel
- **Minimal teaser race cards**: Race name and compelling image only
- **No detailed information**: Designed to encourage click-through to external websites
- "All You Can East Bluffet" featured link: https://www.copperharbortrails.org/bluffet
- Navigation header with "Inferior Timing" logo
- Clean, uncluttered design focusing on visual appeal

### Timing Section (`/timing`)
- Header with "Inferior Timing" logo
- Two main tables:
  - **Active Events**: Current ongoing races
  - **Past Events**: Completed races
- Table columns: Event Name, Date, Number of Participants
- Click on event to navigate to event details

### Event Details Page (`/timing/:eventId`)
- Event information header
- Race selection tabs or dropdown
- Race list with status indicators
- Quick stats (total participants, active races, etc.)

### Race Details Page (`/timing/:eventId/race/:raceId`)
- Race information and status
- Category selection (Overall, Male, Female, Age Groups)
- Multi-tab interface:
  - **Leaderboard Tab**: Primary results display
  - **Race Flow Tab**: Chart visualization of race progress
  - **Statistics Tab**: Additional analytics
- Multi-race comparison capability
- Category filter combinations

### Live Timing Station (`/timing/live/:raceId`)
- Real-time participant tracking
- RFID tag management interface
- Manual entry capabilities
- Offline data storage indicator
- Sync status monitoring
- Participant search and bib lookup

## Race Types and Formats

### Time-Based Races
- Fixed total distance
- Multiple checkpoints at specified distances
- Individual time tracking between checkpoints
- Cumulative time calculation

### Lap-Based Races
- Fixed duration (e.g., 1 hour, 4 hours)
- Shared start/finish line
- Optional intermediate checkpoints within laps
- Lap counting and timing
- Distance calculation based on laps completed

##### RFID Integration

### Multi-Station Architecture
- **Multiple independent checkpoint stations** at different locations
- Each station operates autonomously with local timing capabilities
- **Station Types**:
  - Start/Finish stations (handle both start and finish timing)
  - Intermediate checkpoint stations
  - Mobile/roving stations for special checkpoints
- **Synchronization**: All stations sync to central server when network available
- **Station Identification**: Each station has unique ID for data correlation
- **Conflict Resolution**: Handles overlapping reads and duplicate entries

### Hardware Requirements
- Proxmark3 RFID reader/writer at each station
- USB connectivity to timing station computer/tablet
- Power management for field operations (battery backup)
- Weatherproof housing for outdoor use
- Network connectivity (WiFi/cellular) for sync operations
- Local storage capability for offline operation

### Tag Management
- Write participant UUID to RFID tags
- Associate tags with bib numbers
- Handle tag replacement scenarios
- Support for multiple tag types

### Data Collection
- Real-time tag scanning at checkpoints
- Local storage of timing events
- Offline capability with sync queue
- Network status monitoring
- Automatic sync when connection restored

### Offline Functionality (PWA Approach)

### Progressive Web App Architecture
- **Web-based PWA** with service workers for offline functionality
- **No separate desktop application** required
- Works on tablets, laptops, and mobile devices with modern browsers
- Installable as standalone app on supported devices

### Local Storage Strategy
- **Service Worker Cache**: Static assets and app shell caching
- **IndexedDB**: Local storage for timing data and participant information
- **Local Storage**: Configuration and session data
- **Offline Queue**: Pending timing records stored locally

### Sync Management
- Background sync when network connection restored
- Conflict resolution for duplicate entries across stations
- Data integrity verification before sync
- Manual sync trigger capability for timing operators
- Sync status indicators in UI

### Offline Capabilities
- Complete race timing functionality without internet connection
- Local participant lookup by RFID or bib number
- Real-time leaderboard updates using local data
- Chart and visualization rendering with cached data
- Manual timing entry and participant management

## Charts and Visualizations

### Race Flow Chart (Comprehensive Visualization)
- **Multi-dimensional data visualization** combining:
  - **Position tracking**: Participant position changes over time
  - **Speed analysis**: Speed variations between checkpoints
  - **Checkpoint splits**: Time differences at each checkpoint
  - **Progress indicators**: Visual representation of race progression
- **Interactive features**:
  - Hover tooltips with detailed participant information
  - Click to isolate individual participant tracks
  - Multi-race comparison with contrasting line styles
  - Category filtering that updates chart in real-time
  - Time range selection for detailed analysis
- **Chart types**:
  - Line charts for position over time
  - Bar charts for checkpoint split comparisons
  - Area charts for speed variations
  - Scatter plots for correlation analysis
- **Export capabilities**: PNG, SVG, CSV data export

### Leaderboard Tables
- Sortable columns (position, name, time, category)
- Real-time updates during active races
- Category-specific views
- Export capabilities (CSV, PDF)

### Statistical Charts
- Participant distribution by category
- Finish time distributions
- Checkpoint split analysis
- Historical performance comparisons

## Security and Performance

### Authentication
- JWT-based authentication for admin functions
- Role-based access control (viewer, timer, admin)
- API rate limiting
- CORS configuration for frontend-backend communication

### Data Validation
- Input sanitization for all endpoints
- Race data integrity checks
- Timing record validation
- Participant data consistency verification

### Performance Optimization
- Database indexing on frequently queried fields
- Redis caching for frequently accessed data
- Pagination for large result sets
- WebSocket connections for real-time updates

## Deployment Configuration

### Docker Services
```yaml
services:
  frontend:
    build: ./frontend
    ports: ["3000:3000"]
    environment:
      - VITE_API_URL=http://backend:8080
  
  backend:
    build: ./backend
    ports: ["8080:8080"]
    environment:
      - DB_HOST=postgres
      - DB_NAME=keweenaw_timing
      - REDIS_HOST=redis
    depends_on: [postgres, redis]
  
  postgres:
    image: postgres:14
    environment:
      - POSTGRES_DB=keweenaw_timing
      - POSTGRES_USER=timing_user
      - POSTGRES_PASSWORD=timing_pass
    volumes: ["postgres_data:/var/lib/postgresql/data"]
  
  redis:
    image: redis:7-alpine
    volumes: ["redis_data:/data"]
```

### Google Cloud Configuration
- Cloud SQL instance for PostgreSQL
- Cloud Run services for frontend and backend
- Load balancer for traffic distribution
- Cloud CDN for static asset delivery
- Monitoring and alerting setup

This specification provides the foundation for implementing the Keweenaw Endurance Syndicate race timing system following the constitutional requirements of test-driven development and comprehensive testing coverage.