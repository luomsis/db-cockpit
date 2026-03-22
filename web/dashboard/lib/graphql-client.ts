import { GraphQLClient } from 'graphql-request'

// 开发环境使用模拟token
const DEV_TOKEN = 'dev_tenant:dev_user:admin'

export const graphqlClient = new GraphQLClient('/api/graphql', {
  headers: {
    Authorization: `Bearer ${DEV_TOKEN}`,
  },
})