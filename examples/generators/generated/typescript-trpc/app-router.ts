import { initTRPC } from "@trpc/server";

import { userRouter } from "./auth/routers";

const t = initTRPC.create();

const appRouter = t.router({
  user: userRouter,
});

type AppRouter = typeof appRouter;
