import createMiddleware from "next-intl/middleware";
import { routing } from "./i18n/routing";

export default createMiddleware(routing);

export const config = {
  matcher: ["/((?!api|auth|healthz|readyz|_next|_vercel|.*\\..*).*)"],
};
