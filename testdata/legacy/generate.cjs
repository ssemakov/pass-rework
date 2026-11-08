// Synthetic only. Run with Node12 and the old pinned driver, never a real vault.
const path=require('path'),fs=require('fs');
const root=process.env.PASS_LEGACY_MODULES;
if(!root)throw new Error('Set PASS_LEGACY_MODULES to isolated legacy node_modules');
const knex=require(path.join(root,'knex'));
const filename=path.join(__dirname,'journeyapps-4.0.0.pass');
if(fs.existsSync(filename))throw new Error('Refusing to overwrite fixture');
const db=knex({client:'sqlite3',connection:{filename},useNullAsDefault:true,pool:{min:1,max:1,afterCreate:(conn,done)=>{
 conn.run('PRAGMA cipher_compatibility = 3;');
 conn.run("PRAGMA key = 'public-fixture-password';");
 done();
}}});
(async()=>{
 await db.schema.createTable('login_items',table=>{table.increments('id');table.string('name');table.string('login');table.string('password');table.string('url');table.integer('created_at');table.integer('updated_at');});
 await db('login_items').insert([{id:1,name:"O'Connor 🍋",login:null,password:'public-synthetic-secret\u0000tail',url:'',created_at:1700000000123,updated_at:null},{id:2,name:"O'Connor 🍋",login:'name@example.test',password:'second-public-synthetic-password',url:'https://example.test',created_at:-123,updated_at:1700000000999},{id:31,name:'deleted',password:'deleted synthetic fixture'}]);
 await db('login_items').where('id',31).del();
 const settings={};for(const pragma of ['cipher_version','cipher_page_size','kdf_iter','cipher_use_hmac'])settings[pragma]=await db.raw('PRAGMA '+pragma);
 const rows=await db('login_items').orderBy('id');
 fs.writeFileSync(path.join(__dirname,'expected.json'),JSON.stringify({sourceCommit:'7cd8c6c0bc395fa3676c151a84c1657cda8cb481',driver:'@journeyapps/sqlcipher@4.0.0',knex:'0.20.4',node:process.version,publicPassword:'public-fixture-password',settings,rows},null,2)+'\n');
 await db.destroy();fs.chmodSync(filename,0o600);console.log('Historical synthetic fixture generated and closed.');
})().catch(async()=>{await db.destroy();process.exitCode=1});
