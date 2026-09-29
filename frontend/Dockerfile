# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# NEXT_PUBLIC_* values are inlined into the client bundle at BUILD time by
# Next.js, so they must arrive as build args. docker-compose set them only in the
# runtime stage, which meant API_BASE was compiled as `undefined` and every
# request went to the frontend's own origin, where the rewrite then targets
# localhost:8080 *inside this container* and is refused.
ARG NEXT_PUBLIC_API_URL=http://api.localhost
ARG NEXT_PUBLIC_WS_URL=ws://api.localhost
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_WS_URL=$NEXT_PUBLIC_WS_URL

# Target for the dev-server rewrites in next.config.js. Only used by `npm run
# dev`; the standalone production server does not apply rewrites.
ARG API_URL=http://localhost:8080
ENV API_URL=$API_URL

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# Runtime stage
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
