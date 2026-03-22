/** @type {import('next').NextConfig} */
const nextConfig = {
  rewrites: async () => {
    return {
      beforeFiles: [
        {
          source: '/api/:path*',
          destination: 'http://localhost:8080/api/:path*',
        },
        {
          source: '/events',
          destination: 'http://localhost:8080/events',
        },
      ],
    }
  },
}

export default nextConfig
