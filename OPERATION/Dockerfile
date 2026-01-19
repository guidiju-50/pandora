# Build stage
FROM node:20-alpine AS build

WORKDIR /app

# Build arguments for API URLs
ARG VITE_API_URL=http://localhost:8080/api/v1
ARG VITE_PROCESSING_URL=http://localhost:8081/api/v1
ARG VITE_ANALYSIS_URL=http://localhost:8082/api/v1

# Set environment variables for build
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_PROCESSING_URL=$VITE_PROCESSING_URL
ENV VITE_ANALYSIS_URL=$VITE_ANALYSIS_URL

# Copy package files
COPY package.json ./

# Install dependencies
RUN npm install --legacy-peer-deps

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy custom nginx config
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/

# Copy built files
COPY --from=build /app/dist /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
