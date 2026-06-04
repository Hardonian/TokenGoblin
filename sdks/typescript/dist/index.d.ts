export interface TokenGoblinOptions {
    apiKey?: string;
    baseUrl?: string;
}
export declare class TokenGoblinClient {
    private apiKey;
    private baseUrl;
    constructor(options?: TokenGoblinOptions);
    private request;
    ingestEvent(event: any): Promise<any>;
    ingestBatch(events: any[]): Promise<any>;
    getRecommendations(): Promise<any>;
    getAnomalies(): Promise<any>;
    getWasteReport(): Promise<any>;
    getPromptGraveyard(): Promise<any>;
    getZombieAgents(): Promise<any>;
    getDuplicateClusters(): Promise<any>;
    getCostLeaks(): Promise<any>;
    getHallucinationMap(): Promise<any>;
    getSpendForecast(): Promise<any>;
    getExecutiveScorecard(): Promise<any>;
    getModelComparison(): Promise<any>;
}
