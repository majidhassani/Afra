FROM node:22-alpine AS build

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

ARG VITE_API_BASE_URL=
ARG VITE_ENABLE_MOCKS=false
ARG VITE_DEFAULT_LANGUAGE=fa
ARG VITE_APP_VERSION=production

ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_ENABLE_MOCKS=$VITE_ENABLE_MOCKS
ENV VITE_DEFAULT_LANGUAGE=$VITE_DEFAULT_LANGUAGE
ENV VITE_APP_VERSION=$VITE_APP_VERSION

RUN npm run build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=15s --timeout=5s --retries=5 CMD wget -qO- http://localhost/health || exit 1
