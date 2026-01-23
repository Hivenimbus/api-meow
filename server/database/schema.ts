import { pgTable, text, timestamp, boolean, uuid } from 'drizzle-orm/pg-core'

// Instances table
export const instances = pgTable('instances', {
    id: uuid('id').defaultRandom().primaryKey(),
    name: text('name').notNull(),
    status: text('status').notNull().default('disconnected'),
    phoneNumber: text('phone_number'),
    tagId: text('tag_id'),
    ignoreGroups: boolean('ignore_groups').default(true),
    webhookUrl: text('webhook_url'),
    receiveMessages: boolean('receive_messages').default(true),
    createdAt: timestamp('created_at').defaultNow(),
    updatedAt: timestamp('updated_at').defaultNow()
})

// Tags table
export const tags = pgTable('tags', {
    id: uuid('id').defaultRandom().primaryKey(),
    name: text('name').notNull(),
    color: text('color').notNull(),
    createdAt: timestamp('created_at').defaultNow()
})
