# 🎥 Lightweight Local RTSP-to-MediaMTX Stream Relay Worker (Golang)

Aplikasi relay video stream RTSP ke MediaMTX berbasis **Golang** yang sangat ringan (*low-resource*), berorientasi *event-driven* (MQTT), dan dioptimalkan untuk server lokal (*low-spec*: Debian/Ubuntu di Proxmox LXC, AMD A6, 4GB RAM).

---

## 🌟 Fitur Utama

- ⚡ **Ultra Low-Resource**: Menggunakan subprocess `ffmpeg` mode pass-through bitstream (`-c copy`) via transport TCP tanpa re-encoding video. Penggunaan RAM < 50MB dan CPU < 3% per stream.
- 🔄 **Bootstrap REST API**: Secara otomatis mengambil dan memicu daftar kamera aktif dari endpoint REST API saat startup/reboot.
- 📡 **Real-Time Event-Driven Sync (MQTT)**: Mendengarkan topic `workers/{WORKER_ID}/events` untuk aksi `SYNC_ALL`, `UPSERT_CAMERA`, dan `REMOVE_CAMERA` secara instan tanpa restart worker.
- 🛡️ **Error Isolation & Auto-Reconnect**: Setiap kamera dikelola dalam Goroutine independen (`context.WithCancel`). Jika 1 kamera terputus, retry loop berjalan mandiri tanpa mempengaruhi stream kamera lainnya.
- 🛑 **Graceful Shutdown**: Menangkap sinyal `SIGINT` dan `SIGTERM` untuk mematikan child process FFmpeg dan koneksi MQTT secara bersih.

---

## 📁 Struktur Project

```text
worker-lokal/
├── cmd/
├── internal/
│   ├── client/          # REST API Client untuk bootstrap data kamera
│   ├── config/          # Parser konfigurasi .env & environment variable
│   ├── models/          # Definisi struct Camera, APIResponse, MQTT payloads
│   ├── mqtt/            # MQTT Subscriber & Real-Time event handler
│   └── streamer/        # FFmpeg Goroutine concurrency manager & retry loop
├── .env.example         # Template konfigurasi environment
├── camera-worker.service# Systemd service unit untuk Linux/Proxmox
├── go.mod               # Definisi Go module
├── go.sum               # Hash dependensi Go
├── main.go              # Entry point aplikasi
└── README.md            # Dokumentasi panduan instalasi & build
```

---

## 🛠️ Prasyarat

1. **Golang** (versi 1.20 ke atas)
2. **FFmpeg** terpasang di sistem:
   ```bash
   # Di Debian/Ubuntu / Proxmox LXC
   sudo apt update && sudo apt install -y ffmpeg
   ```

---

## ⚙️ Konfigurasi Environment (`.env`)

Salin file template `.env.example` ke `.env`:

```bash
cp .env.example .env
```

Sesuaikan variabel berikut:

| Variabel | Deskripsi | Contoh |
| :--- | :--- | :--- |
| `WORKER_ID` | Identitas unik worker lokal ini | `worker_cabang_01` |
| `API_BASE_URL` | Endpoint REST API bootstrap kamera | `http://api.domain.com/api/v1/worker/cameras` |
| `API_AUTH_TOKEN` | Bearer token autentikasi API | `eyJhbGciOi...` |
| `MQTT_BROKER` | URL broker MQTT | `tcp://103.xxx.xxx.xxx:1883` |
| `MQTT_CLIENT_ID` | Client ID MQTT (opsional, default: `WORKER_ID`) | `worker_cabang_01` |
| `MQTT_USERNAME` | Username MQTT broker (opsional) | `admin` |
| `MQTT_PASSWORD` | Password MQTT broker (opsional) | `secret` |
| `RETRY_INTERVAL_SECONDS`| Jeda waktu retry saat stream terputus (detik) | `5` |
| `FFMPEG_PATH` | Path binary FFmpeg (opsional, default: `ffmpeg`) | `/usr/bin/ffmpeg` |

---

## 🚀 Panduan Build & Menjalankan

### 1. Menjalankan secara Langsung (Development)

```bash
# Download dependensi
go mod tidy

# Jalankan worker
go run main.go
```

### 2. Build Binary Native

```bash
go build -ldflags="-s -w" -o worker main.go
./worker
```

### 3. Cross-Compile untuk Linux 64-bit (Proxmox LXC / Debian)

Jika Anda melakukan build dari mesin Windows atau macOS untuk dideploy ke server Linux:

```powershell
# PowerShell (Windows)
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"; go build -ldflags="-s -w" -o worker main.go
```

```bash
# Bash (Linux / macOS)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o worker main.go
```

---

## 📦 Deployment sebagai Systemd Service (Linux/Proxmox)

1. Salin binary `worker` dan file `.env` ke direktori target:
   ```bash
   sudo mkdir -p /opt/worker-lokal
   sudo cp worker /opt/worker-lokal/
   sudo cp .env /opt/worker-lokal/
   sudo chmod +x /opt/worker-lokal/worker
   ```

2. Pasang file systemd service:
   ```bash
   sudo cp camera-worker.service /etc/systemd/system/
   sudo systemctl daemon-reload
   ```

3. Aktifkan dan jalankan service:
   ```bash
   sudo systemctl enable --now camera-worker
   ```

4. Cek status dan log secara real-time:
   ```bash
   sudo systemctl status camera-worker
   sudo journalctl -u camera-worker -f
   ```

---

## 📨 Format Payload MQTT Event

Worker akan mendengarkan topic: `workers/{WORKER_ID}/events` (QoS 1).

### 1. Sinkronisasi Penuh (`SYNC_ALL`)
Tarik ulang seluruh daftar kamera dari REST API dan sesuaikan stream aktif:
```json
{
  "action": "SYNC_ALL"
}
```

### 2. Tambah / Perbarui Kamera (`UPSERT_CAMERA`)
Menambah stream kamera baru atau memperbarui URL / status (otomatis restart stream jika konfigurasi berubah):
```json
{
  "action": "UPSERT_CAMERA",
  "camera": {
    "id": "cam_01",
    "name": "Kamera Pos Way Kanan 01",
    "is_active": true,
    "source_url": "rtsp://admin:pass@192.168.10.101:554/stream1",
    "target_url": "rtsp://vps-mediamtx.domain.com:8554/live/waykanan01"
  }
}
```

### 3. Hapus Kamera (`REMOVE_CAMERA`)
Menghentikan dan membersihkan stream kamera tertentu:
```json
{
  "action": "REMOVE_CAMERA",
  "camera_id": "cam_01"
}
```
