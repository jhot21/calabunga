// src/types/api.ts
export interface Calendar {
  id: string;       // CalDAV calendar path e.g. "/dav/alice/personal/"
  name: string;
  color: string;    // hex e.g. "#4285f4"
  ctag: string;
}

export interface RecurrenceEnd {
  type: 'never' | 'date' | 'count';
  value?: string;   // "YYYY-MM-DD" for date, integer string for count
}

export interface Recurrence {
  frequency: 'daily' | 'weekly' | 'monthly' | 'yearly';
  interval: number;
  byDay?: string[];        // ['MO', 'WE', 'FR']
  byMonthDay?: number[];
  end: RecurrenceEnd;
}

export interface Event {
  uid: string;
  calendarId: string;   // CalDAV calendar path
  href?: string;        // CalDAV resource path; edits are written back here
  etag: string;
  title: string;
  start: string;        // UTC ISO8601 e.g. "2026-06-03T10:00:00Z"
  end: string;
  allDay: boolean;
  description?: string;
  location?: string;
  recurrence?: Recurrence;
  originalStart?: string; // update requests only: start of the occurrence that was opened
}

export type EditScope = 'this' | 'thisAndFollowing' | 'all';

export interface ConnectRequest {
  url: string;
  username: string;
  password: string;
  ttl: string;  // "8h" | "24h" | "168h" | "720h"
}

export interface ConnectResponse {
  token: string;
  expiresAt: string;  // RFC3339
}
