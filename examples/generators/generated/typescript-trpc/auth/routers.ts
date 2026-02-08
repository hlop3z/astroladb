import { z } from "zod";
import { initTRPC, TRPCError } from "@trpc/server";

import {
  User,
  UserBase,
  userSchema,
  userBaseSchema,
} from "./schemas";

const t = initTRPC.create();
const router = t.router;
const publicProcedure = t.procedure;

const userRouter = router({
  list: publicProcedure
    .query(async () => {
      throw new TRPCError({ code: "NOT_IMPLEMENTED" });
    }),

  get: publicProcedure
    .input(z.object({ id: z.string().uuid() }))
    .query(async ({ input }) => {
      throw new TRPCError({ code: "NOT_IMPLEMENTED" });
    }),

  create: publicProcedure
    .input(userBaseSchema)
    .mutation(async ({ input }) => {
      throw new TRPCError({ code: "NOT_IMPLEMENTED" });
    }),

  update: publicProcedure
    .input(z.object({ id: z.string().uuid(), data: userBaseSchema }))
    .mutation(async ({ input }) => {
      throw new TRPCError({ code: "NOT_IMPLEMENTED" });
    }),

  delete: publicProcedure
    .input(z.object({ id: z.string().uuid() }))
    .mutation(async ({ input }) => {
      throw new TRPCError({ code: "NOT_IMPLEMENTED" });
    }),
});
