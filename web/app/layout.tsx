import { Bebas_Neue, Space_Grotesk } from "next/font/google"

import "./globals.css"
import { ThemeProvider } from "@/components/theme-provider"
import { cn } from "@/lib/utils"

const grotesk = Space_Grotesk({ subsets: ["latin"], variable: "--font-sans" })

const headline = Bebas_Neue({
  subsets: ["latin"],
  variable: "--font-display",
  weight: "400",
})

const mono = Space_Grotesk({
  subsets: ["latin"],
  variable: "--font-mono",
})

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={cn("antialiased", mono.variable, grotesk.variable, headline.variable)}
    >
      <body>
        <ThemeProvider>{children}</ThemeProvider>
      </body>
    </html>
  )
}
