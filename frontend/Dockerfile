# Build stage
FROM node:22-alpine AS builder

WORKDIR /app

# package.json の packageManager に合わせた pnpm を使う
RUN corepack enable

# Install dependencies
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Copy the source code
COPY . .

# Build the application
RUN pnpm build

# Run stage
FROM node:22-alpine

WORKDIR /app

# Copy the build output from the builder stage
COPY --from=builder /app/.output ./.output

ENV HOST=0.0.0.0
ENV PORT=3000

# Expose the port the app runs on（bff の CORS で許可しているポート）
EXPOSE 3000

# Command to run the application
CMD ["node", ".output/server/index.mjs"]
