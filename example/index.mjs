import {connect} from '@planetscale/database'

const config = {
    url: 'http://root:unused@127.0.0.1:8080'
}

const conn = connect(config)

const results = await conn.transaction(async (tx) => {
  return [
    await tx.execute('select 1'),
    await tx.execute('select 2'),
  ]
})

console.log(results)
