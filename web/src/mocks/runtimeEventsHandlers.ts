/**
 * MSW handlers for runtime events API.
 * Returns mock data for Storybook and tests.
 */
import { create, toJson } from '@bufbuild/protobuf';
import { http, HttpResponse } from 'msw';
import { RuntimeEventV1Schema } from '../pb/proto/scraper/runtime/v1/events_pb';
import { generateMockEvents } from '../test-utils/mockRuntimeEvents';

let cachedEvents = generateMockEvents(20);

export const runtimeEventsHandlers = [
  http.get('*/api/v1/runtime-events', ({ request }) => {
    const url = new URL(request.url);
    const limit = Number(url.searchParams.get('limit') ?? 100);
    const workflowId = url.searchParams.get('workflowId');
    const opId = url.searchParams.get('opId');

    let filtered = cachedEvents;
    if (workflowId) {
      filtered = filtered.filter((e) => e.workflowId === workflowId);
    }
    if (opId) {
      filtered = filtered.filter((e) => e.opId === opId);
    }

    // Serialize with Buf so protobuf JSON matches what fromJson() expects.
    return HttpResponse.json({
      events: filtered
        .slice(0, limit)
        .map((event) => toJson(RuntimeEventV1Schema, create(RuntimeEventV1Schema, event))),
    });
  }),
];
