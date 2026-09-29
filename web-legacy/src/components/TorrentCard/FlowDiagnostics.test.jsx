import axios from 'axios'
import ReactDOM from 'react-dom'
import { act } from 'react-dom/test-utils'

import FlowDiagnostics from './FlowDiagnostics'

jest.mock('axios')
jest.mock('utils/Hosts', () => ({ flowStatusHost: hash => `/flow/status/${hash}` }))

let container

beforeEach(() => {
  jest.useFakeTimers()
  axios.get.mockReset()
  container = document.createElement('div')
  document.body.appendChild(container)
})

afterEach(() => {
  act(() => {
    ReactDOM.unmountComponentAtNode(container)
  })
  container.remove()
  jest.clearAllTimers()
  jest.useRealTimers()
})

test('uses the API file index and distinguishes unknown buffer estimates', async () => {
  axios.get.mockResolvedValue({ data: { sessions: [{ group: 'phone', file_index: 1, state: 'PLAYING' }] } })
  await act(async () => {
    ReactDOM.render(<FlowDiagnostics hash='test' onClose={() => {}} />, container)
  })
  expect(document.body.textContent).toContain('File 1')
  expect(document.body.textContent).not.toContain('File 2')
  expect(document.body.textContent).toContain('Estimated server buffer aheadUnknown')
  expect(document.body.textContent).toContain('Buffer riskUnknown')
})

test('waits for a response before polling again and aborts on close', async () => {
  let resolve
  axios.get.mockImplementation(
    () =>
      new Promise(done => {
        resolve = done
      }),
  )
  await act(async () => {
    ReactDOM.render(<FlowDiagnostics hash='test' onClose={() => {}} />, container)
  })
  act(() => jest.advanceTimersByTime(10000))
  expect(axios.get).toHaveBeenCalledTimes(1)
  const request = axios.get.mock.calls[0][1]
  expect(request.timeout).toBe(5000)
  await act(async () => resolve({ data: { sessions: [] } }))
  act(() => jest.advanceTimersByTime(2000))
  expect(axios.get).toHaveBeenCalledTimes(2)
  act(() => {
    ReactDOM.unmountComponentAtNode(container)
  })
  expect(request.signal.aborted).toBe(true)
  await act(async () => resolve({ data: { sessions: [] } }))
  act(() => jest.advanceTimersByTime(10000))
  expect(axios.get).toHaveBeenCalledTimes(2)
})
