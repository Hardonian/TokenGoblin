import "@testing-library/jest-dom";
import { TextEncoder, TextDecoder } from 'util';
import { ReadableStream } from 'stream/web';
import { MessageChannel, MessagePort } from 'worker_threads';

global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;
global.ReadableStream = ReadableStream;
global.MessageChannel = MessageChannel;
global.MessagePort = MessagePort;

// Mock IntersectionObserver
class IntersectionObserver {
  observe = jest.fn();
  disconnect = jest.fn();
  unobserve = jest.fn();
}

Object.defineProperty(window, 'IntersectionObserver', {
  writable: true,
  configurable: true,
  value: IntersectionObserver
});

Object.defineProperty(global, 'IntersectionObserver', {
  writable: true,
  configurable: true,
  value: IntersectionObserver
});

// Mock Request and Response for next/server polyfills
const { Request, Response, Headers } = require('undici');
global.Request = Request;
global.Response = Response;
global.Headers = Headers;
