import { z } from 'zod';

export const ProtocolCategorySchema = z.enum(['base', 'standalone', 'wrapper']);

export const ProtocolStatusSchema = z.enum(['running', 'stopped', 'error', 'installing', 'unknown']);

export const ProtocolDetailedSchema = z.object({
  id: z.string(),
  name: z.string(),
  category: ProtocolCategorySchema,
  description: z.string(),
  source: z.string(),
  xrayNative: z.boolean(),
  status: ProtocolStatusSchema,
  healthy: z.boolean(),
  port: z.number().optional(),
  installed: z.boolean().optional(),
  serviceName: z.string().optional(),
  supportedProtocols: z.array(z.string()).optional(),
});
