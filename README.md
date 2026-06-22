# Hollow

## Getting Started

Make sure [Bun](https://bun.sh/) is installed on your system.

### Installation

```sh
git clone https://github.com/haiderking1/Hollow.git
cd Hollow
bun install
```

### Running in Development

To run the Vite web build and launch Electron concurrently:

```sh
bun run electron:dev
```

To run only the Vite client:

```sh
bun run dev
```

To run only the TS runtime bridge:

```sh
bun run start
```

### Building for Production

To build the desktop production package:

```sh
bun run build:desktop
```

### Running Tests & Checks

- **Typecheck**: `bun run typecheck`
- **Unit Tests**: `bun test`
