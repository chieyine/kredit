export type FeeTerms={policy_revision:number;base_bps:number;collection_bps:number};
export function feeDisclosure(terms?:FeeTerms|null){
 const base=terms?.base_bps??50,collection=terms?.collection_bps??50;
 if(!Number.isInteger(base)||!Number.isInteger(collection)||base<0||base>1000||collection<0||collection>1000)return 'Fee terms are unavailable. Please refresh before accepting.';
 return `The seller pays ${base/100}% when this sale becomes active, plus ${collection/100}% on any amount Kredit successfully collects after the permitted collection time. These fees are not added to the buyer’s principal.`;
}
