import { Queue } from '../types/queues'

export const getQueues = async () => {
  const response = await fetch('/API/v1/queues')
  return await response.json()
}

export const getQueue = async (name: string) : Promise<Queue> => {
  const response = await fetch(`/API/v1/queues/${name}`)
  const data = await response.json()

  return data
}
