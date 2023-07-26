-- create a db per blockchain network (ethereum or moonbeam, mainnet or goerli or testnet)
-- and development stage (prod or dev)
-- like ethereum_goerli2023_dev or moonbeam_mainnet_prod

-- drop database ${evm_chain}_${evm_network}${evm_shard}_${namespace};
create database ${evm_chain}_${evm_network}${evm_shard}_${namespace};

create user evm_archive with password '${db_password_evm_archive}';

grant connect, create on database ${evm_chain}_${evm_network}${evm_shard}_${namespace} to evm_archive;
