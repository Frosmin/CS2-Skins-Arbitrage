# CS2 Skins Arbitrage Scanner 🎯📈

Una aplicación web full-stack diseñada para identificar oportunidades de **arbitraje y gangas en skins de Counter-Strike 2 (CS2)**, comparando en tiempo real las publicaciones del mercado de **[CSFloat](https://csfloat.com/)** contra los precios de referencia de **Steam**.

---

## 📑 Tabla de Contenidos

- [Descripción General](#-descripción-general)
- [¿Cómo Funciona el Sistema?](#-cómo-funciona-el-sistema)
  - [Lógica de Detección de Arbitraje](#lógica-de-detección-de-arbitraje)
  - [Métricas y Fórmulas Clave](#métricas-y-fórmulas-clave)
  - [Arquitectura de la Aplicación](#arquitectura-de-la-aplicación)
- [Requisitos Previos (¿Qué cosas se necesitan?)](#-requisitos-previos-qué-cosas-se-necesitan)
  - [Software Necesario](#software-necesario)
  - [Credenciales Requeridas](#credenciales-requeridas)
- [Guía de Instalación y Ejecución](#-guía-de-instalación-y-ejecución)
  - [1. Clonar el repositorio](#1-clonar-el-repositorio)
  - [2. Configurar el Backend (Go)](#2-configurar-el-backend-go)
  - [3. Iniciar el Backend](#3-iniciar-el-backend)
  - [4. Configurar e Iniciar el Frontend (Angular)](#4-configurar-e-iniciar-el-frontend-angular)
- [Estructura del Repositorio](#-estructura-del-repositorio)
- [Documentación del API Backend](#-documentación-del-api-backend)
  - [Endpoint Principal: `GET /api/listings`](#endpoint-principal-get-apilistings)
- [Ejecución de Pruebas (Testing)](#-ejecución-de-pruebas-testing)
- [Solución de Problemas Comunes (FAQ)](#-solución-de-problemas-comunes-faq)

---

## 🔍 Descripción General

En el mercado de skins de CS2, los precios fluctúan constantemente entre diferentes plataformas. **CS2 Skins Arbitrage** automatiza la búsqueda de publicaciones donde el precio de compra directa en **CSFloat** es significativamente inferior al valor de referencia del mercado de la comunidad de **Steam**.

La plataforma permite a los usuarios:
- Filtrar por rangos de precio en USD (ej. skins económicas de \$0.03 a \$1.00 o rangos personalizados).
- Identificar el porcentaje de descuento real y la ganancia bruta estimada.
- Visualizar el nivel de desgaste exacto (*Wear / Float Value*) de cada skin en una barra gráfica interactiva clasificada por categorías (FN, MW, FT, WW, BS).
- Descartar distorsiones de precio causadas por pegatinas (*stickers*), llaveros o etiquetas de nombre mediante el filtro **Item Factor**.
- Acceder directamente al enlace de compra de cada artículo en CSFloat con un solo clic.

---

## ⚙️ ¿Cómo Funciona el Sistema?

### Lógica de Detección de Arbitraje

1. **Consulta a la API de CSFloat**: El backend solicita las publicaciones activas tipo `buy_now` (compra inmediata) para la categoría `1` (CS2).
2. **Paginación Inteligente**: La API de CSFloat limita las respuestas a lotes de 50 artículos por petición. Si el usuario solicita un límite mayor (hasta 200), el backend maneja automáticamente la paginación por *cursor* de forma transparente.
3. **Normalización y Conversión Monetaria**: La API de CSFloat maneja precios en centavos enteros (`int64`). El backend convierte todos los montos a dólares estadounidenses (`USD`), redondeando a 2 decimales.
4. **Filtrado de Rentabilidad**: Se descartan automáticamente los artículos donde el precio de referencia de Steam sea menor o igual al precio de CSFloat (sin margen de ganancia).
5. **Detección y Filtro de "Item Factor"**:
   - CSFloat calcula un `predicted_price` que considera el sobreprecio por calcomanías (*stickers*), patrones o rarezas.
   - `Item Factor = predicted_price - base_price`.
   - Cuando el filtro `only_no_factor=true` está activado, se ignoran aquellas skins cuyo precio esté inflado por pegatinas, evitando comprar skins cuyo valor estimado dependa de stickers difíciles de revender.
6. **Ordenamiento por Oportunidad**: Las ofertas se ordenan de mayor a menor según el porcentaje de descuento (`best_deal`), permitiendo visualizar primero las mejores gangas.

### Métricas y Fórmulas Clave

| Métrica | Definición / Fórmula |
| :--- | :--- |
| **Precio CSFloat** | Precio de venta listado por el vendedor (`price / 100`). |
| **Precio Referencia Steam** | Precio base de mercado en Steam (`reference.base_price / 100`). |
| **Descuento (%)** | `((Precio_Steam - Precio_CSFloat) / Precio_Steam) * 100` |
| **Ganancia Bruta Estimada** | `max(Precio_Steam - Precio_CSFloat, 0)` |
| **Item Factor** | `(predicted_price - base_price) / 100` |
| **Float / Wear** | Valor decimal entre `0.00` y `1.00` que define el desgaste físico de la skin. |

#### Clasificación de Estados de Desgaste (Wear):
- **FN (Factory New / Recién Fabricado)**: `0.00` - `0.07`
- **MW (Minimal Wear / Casi Nuevo)**: `0.07` - `0.15`
- **FT (Field-Tested / Algo Desgastado)**: `0.15` - `0.38`
- **WW (Well-Worn / Bastante Desgastado)**: `0.38` - `0.45`
- **BS (Battle-Scarred / Deplorable)**: `0.45` - `1.00`

### Arquitectura de la Aplicación

```text
  [ Navegador Web ]
         │
         ▼  (HTTP / Angular Signals)
  ┌────────────────────────────────────────┐
  │  Frontend: Angular 21 (Port 4200)      │
  │  - Dashboard interactivo de filtros    │
  │  - Tarjetas de oportunidades y float   │
  └──────────────────┬─────────────────────┘
                     │  GET /api/listings (CORS)
                     ▼
  ┌────────────────────────────────────────┐
  │  Backend: Go + Gin Engine (Port 8080)  │
  │  - Validación de filtros y cursor      │
  │  - Lógica de arbitraje y Item Factor   │
  └──────────────────┬─────────────────────┘
                     │  GET /api/v1/listings (Authorization Header)
                     ▼
  ┌────────────────────────────────────────┐
  │         API Pública de CSFloat         │
  └────────────────────────────────────────┘
```

---

## 📦 Requisitos Previos (¿Qué cosas se necesitan?)

Para ejecutar este proyecto en tu entorno local necesitas tener instalado lo siguiente:

### Software Necesario

1. **Go (Golang)**:
   - Versión **1.22 o superior** (el proyecto está configurado con Go `1.25.x`).
   - Puedes verificarlo con: `go version`
   - [Descargar Go](https://go.dev/dl/)
2. **Node.js**:
   - Versión **18.x LTS o superior** (recomendado Node 20.x o 24.x).
   - Puedes verificarlo con: `node -v`
   - [Descargar Node.js](https://nodejs.org/)
3. **NPM**:
   - Administrador de paquetes de Node (incluido con Node.js).
   - Puedes verificarlo con: `npm -v`
4. **Git**:
   - Para clonar y gestionar el repositorio.

### Credenciales Requeridas

1. **API Key de CSFloat**:
   - Es **indispensable** para consultar las publicaciones en CSFloat.
   - **¿Cómo obtenerla?**:
     1. Inicia sesión en [CSFloat](https://csfloat.com/) con tu cuenta de Steam.
     2. Ve a la configuración de tu perfil / sección de desarrolladores (**Settings -> Developer / API Keys**).
     3. Genera una nueva clave de API y cópiala.

---

## 🚀 Guía de Instalación y Ejecución

Sigue estos pasos en orden para poner a correr la aplicación completa:

### 1. Clonar el repositorio

Abre una terminal y clona el proyecto:

```bash
git clone https://github.com/Frosmin/cs2-skins-arbitrage.git
cd cs2-skins-arbitrage
```

---

### 2. Configurar el Backend (Go)

1. Ingresa a la carpeta `backend`:
   ```bash
   cd backend
   ```

2. Crea o edita el archivo de variables de entorno `.env` en la carpeta `backend/`. Puedes basarte en el archivo `.env.example`:
   ```bash
   # En Windows PowerShell:
   Copy-Item .env.example .env

   # En Linux / macOS / Git Bash:
   cp .env.example .env
   ```

3. Abre el archivo `backend/.env` y coloca tu clave de CSFloat:
   ```env
   CSFLOAT_API_KEY=tu_api_key_de_csfloat_aqui
   ```

4. Descarga las dependencias de Go:
   ```bash
   go mod download
   ```

---

### 3. Iniciar el Backend

Dentro del directorio `backend/`, ejecuta:

```bash
go run ./cmd/api
```

El servidor HTTP se iniciará en el puerto **`8080`**:
```text
[GIN-debug] Listening and serving HTTP on :8080
```

> **Nota:** Puedes comprobar que responde accediendo a `http://localhost:8080/api/listings` desde tu navegador o mediante un cliente HTTP como cURL o Postman.

---

### 4. Configurar e Iniciar el Frontend (Angular)

1. Abre una **nueva ventana o pestaña de terminal** (deja el backend corriendo en la anterior).
2. Desde la raíz del proyecto, navega a la carpeta de la interfaz:
   ```bash
   cd frontend/arbitrage-interface
   ```

3. Instala los paquetes y dependencias de Node:
   ```bash
   npm install
   ```

4. Inicia el servidor de desarrollo de Angular:
   ```bash
   npm start
   ```
   *(O alternativamente: `npx ng serve`)*

5. Una vez compilado exitosamente, abre tu navegador web favorito y entra a:
   ```
   http://localhost:4200/
   ```

¡Listo! Ya puedes ver el panel interactivo, ajustar filtros de precio, buscar gangas y ordenar por mejor porcentaje de descuento.

---

## 📂 Estructura del Repositorio

```text
CS2-Skins-Arbitrage/
│
├── README.md                          # Documentación general del proyecto (este archivo)
├── .gitignore                         # Archivos ignorados por Git
│
├── backend/                           # Servidor API desarrollado en Go
│   ├── .env                           # Variables de entorno locales (con CSFLOAT_API_KEY)
│   ├── .env.example                   # Plantilla de ejemplo de variables de entorno
│   ├── go.mod                         # Definición del módulo y dependencias Go
│   ├── go.sum                         # Checksums de dependencias Go
│   ├── cmd/
│   │   └── api/
│   │       └── main.go                # Punto de entrada (Gin server, CORS y rutas)
│   └── csFloat/
│       ├── handler.go                 # Controladores HTTP y parsing de parámetros
│       ├── handler_test.go            # Pruebas unitarias de los controladores
│       ├── service.go                 # Lógica de consumo de CSFloat, filtrado y cálculo
│       └── service_test.go            # Pruebas unitarias del servicio de arbitraje
│
└── frontend/
    └── arbitrage-interface/           # SPA desarrollada en Angular 21
        ├── package.json               # Dependencias de npm y scripts
        ├── angular.json               # Configuración del workspace de Angular
        └── src/
            └── app/
                ├── app.routes.ts      # Enrutamiento de la aplicación
                ├── data/
                │   └── listings-api.ts# Servicio HTTP para comunicarse con el backend Go
                ├── types/
                │   └── listings.ts    # Modelos TypeScript e interfaces de datos
                └── pages/
                    └── prices-list/   # Componente principal de listado y filtros
                        ├── prices-list.html
                        ├── prices-list.scss
                        └── prices-list.ts
```

---

## 📡 Documentación del API Backend

### Endpoint Principal: `GET /api/listings`

Consulta oportunidades filtradas y procesadas desde CSFloat.

#### Parámetros de Consulta (Query Params)

| Parámetro | Tipo | Requerido | Valor por Defecto | Descripción |
| :--- | :--- | :--- | :--- | :--- |
| `min_price` | `float` | No | `0.03` | Precio mínimo en dólares (USD). |
| `max_price` | `float` | No | `1.00` | Precio máximo en dólares (USD). |
| `limit` | `int` | No | `50` | Cantidad de artículos a obtener (máximo `200`). |
| `sort` | `string` | No | `best_deal` | Criterio de orden: `best_deal`, `lowest_price`, `most_recent`. |
| `only_no_factor` | `bool` | No | `true` | Si es `true`, descarta ítems con sobreprecio por stickers/aditamentos. |

#### Ejemplo de Petición (cURL)

```bash
curl "http://localhost:8080/api/listings?min_price=0.10&max_price=2.00&limit=10&only_no_factor=true"
```

#### Ejemplo de Respuesta (JSON)

```json
{
  "items": [
    {
      "id": "123456789012345678",
      "market_hash_name": "AK-47 | Safari Mesh (Field-Tested)",
      "wear": 0.2241589,
      "icon_url": "https://community.cloudflare.steamstatic.com/economy/image/...",
      "csfloat_price": 0.45,
      "steam_reference_price": 0.65,
      "predicted_price": 0.65,
      "item_factor": 0.00,
      "discount_percent": 30.77,
      "purchase_url": "https://csfloat.com/item/123456789012345678"
    }
  ],
  "filters": {
    "min_price": 0.1,
    "max_price": 2,
    "limit": 10,
    "sort": "best_deal",
    "only_no_factor": true
  },
  "count": 1
}
```

---

## 🧪 Ejecución de Pruebas (Testing)

### Pruebas del Backend (Go)

Para ejecutar los tests unitarios del backend:

```bash
cd backend
go test -v ./...
```

### Pruebas del Frontend (Angular + Vitest)

Para ejecutar la suite de pruebas unitarias del frontend:

```bash
cd frontend/arbitrage-interface
npm test
```

---

## 🛠️ Solución de Problemas Comunes (FAQ)

### 1. `Error 500: la variable de entorno CSFLOAT_API_KEY está vacía`
- **Causa**: El backend no encuentra la variable `CSFLOAT_API_KEY`.
- **Solución**: Asegúrate de tener el archivo `.env` dentro de la carpeta `backend/` con el contenido `CSFLOAT_API_KEY=tu_token_aqui`. Verifica que no tenga espacios extra alrededor del `=`.

### 2. Error de conexión con CSFloat (`Status 401 Unauthorized`)
- **Causa**: Tu API Key de CSFloat es inválida o ha expirado.
- **Solución**: Inicia sesión en CSFloat, genera una nueva API Key y actualízala en `backend/.env`. Reinicia el backend (`go run ./cmd/api`).

### 3. Error de CORS en el navegador (`Blocked by CORS policy`)
- **Causa**: El frontend se ejecuta en un host/puerto distinto al configurado en el middleware de Gin.
- **Solución**: Por defecto, el backend permite peticiones desde `http://localhost:4200`. Si ejecutas el frontend en otro puerto, edita `backend/cmd/api/main.go` en la función `corsMiddleware` para permitir tu origen.

### 4. Puerto en uso (`bind: address already in use` o `Port 8080 is already in use`)
- **Causa**: Otra instancia del backend u otro programa ya está usando el puerto 8080 (o 4200 para Angular).
- **Solución**: Cierra el proceso que ocupa el puerto o cámbialo en `backend/cmd/api/main.go` (ej. `:8081`) y actualiza `baseUrl` en `frontend/arbitrage-interface/src/app/data/listings-api.ts`.
