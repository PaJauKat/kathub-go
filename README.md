# KatHub (Wails v2 + Go + Tailwind CSS)

KatHub migrado de WPF (.NET 8) a **Wails v2** con backend en **Go** e interfaz moderna en **HTML5 + Tailwind CSS + Vanilla JS**.

## 🚀 Estructura del Proyecto

```text
KatHub/
├── app.go                      # Estructura principal App y métodos expuestos a Wails
├── main.go                     # Punto de entrada Wails y configuración de ventana
├── go.mod                      # Dependencias de Go (Wails v2)
├── wails.json                  # Configuración de Wails v2
├── pkg/
│   ├── katplugins/             # Lógica de detección, crackeo (EthanVann), descarga e instalación JAR
│   │   ├── service.go
│   │   └── EthanVannInstaller.jar (Embed)
│   ├── lolconfig/              # Locker de PersistedSettings.json (League of Legends)
│   │   └── service.go
│   └── updater/                # Verificación y descarga de releases desde GitHub (PaJauKat/KatHub)
│       └── service.go
├── frontend/
│   ├── src/
│   │   ├── index.html          # UI traducida de MainWindow.xaml con Tailwind CSS
│   │   ├── main.js             # Integración con window.go.main.App.*
│   │   └── assets/             # Iconos e imágenes
│   └── dist/                   # Assets compilados embebidos en el binario final
└── build/
    └── appicon.ico             # Icono de la aplicación
```

## 🛠️ Requisitos

- [Go 1.21+](https://golang.org/)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)
  ```bash
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
  ```

## 💻 Desarrollo y Compilación

### Ejecutar en modo desarrollo con live reload:
```bash
wails dev
```

### Compilar binario de producción optimizado (Windows x64):
```bash
wails build -clean -platform windows/amd64 -windowsconsole=false
```

### Crear Instalador
Deja el ejecutable en `%LOCALAPPDATA%\KatHub`
```bash
.\crearexe.ps1
```

### Subir nueva version 
```
git tag vX.X.X
git push origin vX.X.X
```

El ejecutable resultante estará disponible en `build/bin/KatHub.exe` y el instalador en `build/bin/KatHub_Setup.exe`
