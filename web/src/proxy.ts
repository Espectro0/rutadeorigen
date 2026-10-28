import createMiddleware from "next-intl/middleware";
import { routing } from "./i18n/routing";

// Next.js 16 renombró "middleware" a "proxy".
export default createMiddleware(routing);

export const config = {
  // Todo menos api, auth, sondas, _next, _vercel y archivos con extensión (favicon.ico, etc.)
  matcher: ["/((?!api|auth|healthz|readyz|_next|_vercel|.*\\..*).*)"],
};
